package htun

import (
	"bufio"
	"bytes"
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
	"strconv"
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
			Dial: func(network, addr string) (conn net.Conn, err error) {
				if self.ServerHost != "" {
					addr = self.ServerHost
				}

				return net.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{KeyLogWriter: self.KeyLogWriter, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS13, InsecureSkipVerify: true},
		},
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// 	return http.ErrUseLastResponse
		// },
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

	// log.Println(localConn.RemoteAddr(), https)

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
		err error

		resp     *http.Response
		reqBytes []byte

		hostport = req.Host
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if len(strings.Split(hostport, ":")) != 2 {
		if https {
			hostport += ":443"
		} else {
			hostport += ":80"
		}
	}

	if self.Verbose {
		if https {
			req.URL.Scheme = "https"
		} else {
			req.URL.Scheme = "http"
		}
		req.URL.Host = req.Host
		log.Println(req.URL)
	}

	req.Header.Set("Connection", "close")

	if reqBytes, err = httputil.DumpRequestOut(req, true); err != nil {
		return
	}

	// fmt.Println(string(reqBytes))

	reqBytes = compress(reqBytes)

	if req, err = http.NewRequest(http.MethodPost, self.ServerAddr, bytes.NewReader(reqBytes)); err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", strconv.Itoa(len(reqBytes)))
	// req.Header.Set("User-Agent", "")

	req.Header.Add("xhost", hostport)
	if https {
		req.Header.Add("xprotocol", "https")
	} else {
		req.Header.Add("xprotocol", "http")
	}

	if resp, err = self.HttpClient.Do(req); err != nil {
		return
	}
	// defer resp.Body.Close()

	if _, err = io.Copy(localConn, resp.Body); err != nil {
		return
	}

	self.handleConn(localConn, https)
}
