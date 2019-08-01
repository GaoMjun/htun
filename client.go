package htun

import (
	"bufio"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tencentyun/scf-go-lib/cloudevents/scf"
)

var urlParse = url.Parse

type Client struct {
	LocalAddr  string
	ServerAddr string
	ServerHost string
	HttpClient *http.Client
	CA         *x509.Certificate
	PK         *rsa.PrivateKey
	Key        []byte
	Verbose    bool
}

func ClientRun(args []string) (err error) {
	flags := flag.NewFlagSet("client", flag.PanicOnError)
	addr := flags.String("l", ":80", "http server listen address")
	socksaddr := flags.String("sl", "", "socks5 server listen address")
	pass := flags.String("k", "", "password")
	sa := flags.String("sa", "", "server http address")
	sh := flags.String("sh", "", "server http host")
	capath := flags.String("ca", "", "certificate file")
	pkpath := flags.String("pk", "", "private key file")
	verbose := flags.Bool("v", false, "verbose mode")
	flags.Parse(args)

	var (
		ca *x509.Certificate
		pk *rsa.PrivateKey
	)

	if ca, pk, err = LoadCert(*capath, *pkpath); err != nil {
		return
	}

	client := Client{
		LocalAddr:  *addr,
		ServerAddr: *sa,
		ServerHost: *sh,
		CA:         ca,
		PK:         pk,
		Key:        []byte(*pass),
		Verbose:    *verbose,
	}

	if len(*socksaddr) > 0 {
		go socksServerRun(*socksaddr, client.handleConn)
	}

	err = client.Run()

	return
}

func (self *Client) Run() (err error) {
	self.HttpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 128,
			MaxConnsPerHost:     0,
			Dial: func(network, addr string) (conn net.Conn, err error) {
				if self.ServerHost != "" {
					addr = self.ServerHost
				}

				return net.Dial(network, addr)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var (
		l net.Listener
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if l, err = net.Listen("tcp", self.LocalAddr); err != nil {
		return
	}

	for {
		if conn, err := l.Accept(); err == nil {

			go self.handleConn(conn, false)
			continue
		}
	}
}

func (self *Client) handleConn(localConn net.Conn, https bool) {
	defer localConn.Close()

	var (
		err error
		req *http.Request
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	if req, err = http.ReadRequest(bufio.NewReader(localConn)); err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
			return
		}

		if req.URL.Port() == "80" {
			self.handleConn(localConn, false)
			return
		}

		tlsConfig := &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return Cert(info.ServerName, self.CA, self.PK)
			}}
		localConn = tls.Server(localConn, tlsConfig)
		self.handleConn(localConn, true)
		return
	}

	self.forwardRequest(localConn, req, https)
}

func (self *Client) forwardRequest(localConn net.Conn, req *http.Request, https bool) {
	var (
		err        error
		resp       *http.Response
		bodyBytes  []byte
		bodyString string
		gwResp     = &scf.APIGatewayProxyResponse{}
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	req.Header.Add("xhost", req.Host)
	if https {
		req.Header.Add("xprotocol", "https")
	} else {
		req.Header.Add("xprotocol", "http")
	}

	if self.Verbose {
		var s string
		if https {
			s = "https://" + req.Host + req.RequestURI
		} else {
			s = "http://" + req.Host + req.RequestURI
		}

		log.Println(s)
	}

	if strings.HasPrefix(req.RequestURI, "http") {
		req.RequestURI = req.RequestURI[len("http://"+req.Host):]
	}

	if req.URL, err = urlParse(self.ServerAddr + req.RequestURI); err != nil {
		return
	}

	req.RequestURI = ""
	req.Host = req.URL.Host

	if resp, err = self.HttpClient.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	bodyBytes, err = ioutil.ReadAll(resp.Body)

	if len(bodyBytes) <= 0 {
		return
	}

	if bodyString, err = strconv.Unquote(string(bodyBytes)); err != nil {
		return
	}
	bodyBytes = []byte(bodyString)

	if err = json.Unmarshal(bodyBytes, gwResp); err != nil {
		return
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d OK\n", gwResp.StatusCode)
	headers := ""
	for k, v := range gwResp.Headers {
		headers += k + ": " + v + "\n"
	}
	headers += "\n"
	body := []byte{}

	if len(gwResp.Body) > 0 {
		if body, err = base64.StdEncoding.DecodeString(gwResp.Body); err != nil {
			return
		}
	}

	if _, err = localConn.Write([]byte(statusLine)); err != nil {
		return
	}
	if _, err = localConn.Write([]byte(headers)); err != nil {
		return
	}
	if _, err = localConn.Write(body); err != nil {
		return
	}
}
