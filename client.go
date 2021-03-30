package htun

import (
	"bufio"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var urlParse = url.Parse

type Client struct {
	LocalAddr                     string
	ServerAddr                    string
	ServerHost                    string
	HttpClient, DefaultHttpClient *http.Client
	CA                            *x509.Certificate
	PK                            *rsa.PrivateKey
	Key                           []byte
	Verbose                       bool
	KeyLogWriter                  io.Writer
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
	sslkeylogfile := flags.String("sslkeylogfile", "", "sslkeylogfile path")
	flags.Parse(args)

	var (
		ca           *x509.Certificate
		pk           *rsa.PrivateKey
		keyLogWriter io.Writer
	)

	if ca, pk, err = LoadCert(*capath, *pkpath); err != nil {
		return
	}

	if len(*sslkeylogfile) > 0 {
		if keyLogWriter, err = os.OpenFile(*sslkeylogfile, os.O_WRONLY, os.ModePerm); err != nil {
			return
		}
	}

	client := Client{
		LocalAddr:    *addr,
		ServerAddr:   *sa,
		ServerHost:   *sh,
		CA:           ca,
		PK:           pk,
		Key:          []byte(*pass),
		Verbose:      *verbose,
		KeyLogWriter: keyLogWriter,
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
			// MaxIdleConns:        0,
			// MaxIdleConnsPerHost: 128,
			// MaxConnsPerHost:     0,
			Dial: func(network, addr string) (conn net.Conn, err error) {
				if self.ServerHost != "" {
					addr = self.ServerHost
				}

				return net.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{KeyLogWriter: self.KeyLogWriter, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS13, InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	self.DefaultHttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
		if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
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
		respBytes  []byte
		requestURI = req.RequestURI
		httpClient *http.Client
		serverAddr string
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	req.Header.Add("X-Accept-Encoding", req.Header.Get("Accept-Encoding"))
	req.Header.Add("xhost", req.Host)
	if https {
		req.Header.Add("xprotocol", "https")
	} else {
		req.Header.Add("xprotocol", "http")
	}

	if self.Verbose {
		if https {
			req.URL.Scheme = "https"
		} else {
			req.URL.Scheme = "http"
		}
		req.URL.Host = req.Host
		log.Println(req.URL)

		// var bs []byte
		// if bs, err = httputil.DumpRequest(req, true); err != nil {
		// 	return
		// }

		// fmt.Print(string(bs))
	}

	if strings.HasPrefix(req.RequestURI, "http") {
		requestURI = req.RequestURI[len("http://"+req.Host):]
	}

	req.RequestURI = ""

	// if strings.HasSuffix(req.Host, ".googlevideo.com") {
	// 	httpClient = self.DefaultHttpClient
	// 	serverAddr = "https://googlevideo.bigbuckbunny.workers.dev"
	// }

	// if strings.HasSuffix(req.Host, "github.com") {
	// 	httpClient = self.DefaultHttpClient
	// 	serverAddr = "https://github.bigbuckbunny.workers.dev"
	// }

	if serverAddr == "" {
		serverAddr = self.ServerAddr
	}

	if httpClient == nil {
		httpClient = self.HttpClient
	}

	if req.URL, err = urlParse(serverAddr + requestURI); err != nil {
		return
	}

	req.Host = req.URL.Host

	if resp, err = httpClient.Do(req); err != nil {
		return
	}

	resp.TransferEncoding = nil
	if contentEncoding := resp.Header.Get("X-Content-Encoding"); len(contentEncoding) > 0 {
		resp.Header.Set("Content-Encoding", contentEncoding)
	}

	// resp.Header.Del("Content-Disposition")

	if respBytes, err = httputil.DumpResponse(resp, false); err != nil {
		return
	}

	if _, err = localConn.Write(respBytes); err != nil {
		return
	}

	_, err = io.Copy(localConn, resp.Body)
}
