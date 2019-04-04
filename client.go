package htun

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/golang/snappy"

	"github.com/GaoMjun/ladder"
)

type Client struct {
	LocalAddr  string
	ServerAddr string
	CA         *x509.Certificate
	PK         *rsa.PrivateKey
}

func (self *Client) Run() (err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", self.LocalAddr); err != nil {
		return
	}
	defer l.Close()

	log.Println("client run at", l.Addr())

	for {
		conn, _ := l.Accept()

		go self.handleConn(conn, false)
	}
}

func (self *Client) handleConn(localConn net.Conn, https bool) {
	var (
		err     error
		request *ladder.Request
	)
	defer func() {
		localConn.Close()

		if err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	if request, err = ladder.NewRequest(localConn); err != nil {
		return
	}

	if request.HttpRequest.Method == http.MethodConnect {
		if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
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

	// fmt.Print(request.Dump())

	var (
		u          *url.URL
		rawReq     []byte
		tunnelConn net.Conn
		ip         = "149.129.62.126"
	)
	if u, err = url.Parse(self.ServerAddr); err != nil {
		return
	}
	request.HttpRequest.Host = u.Host
	request.HttpRequest.RequestURI = "/" + base64.StdEncoding.EncodeToString([]byte(request.HttpRequest.RequestURI))
	request.HttpRequest.Header.Del("Origin")
	request.HttpRequest.Header.Del("Referer")
	request.HttpRequest.Header.Add("Real-Reaquest", base64.StdEncoding.EncodeToString(snappy.Encode(nil, request.Bytes())))
	if https {
		request.HttpRequest.Header.Add("Https", "true")
	}

	if rawReq, err = httputil.DumpRequest(request.HttpRequest, false); err != nil {
		return
	}

	if tunnelConn, err = DialHttp(self.ServerAddr, ip); err != nil {
		return
	}
	defer tunnelConn.Close()

	if _, err = tunnelConn.Write(rawReq); err != nil {
		return
	}

	fmt.Println(string(rawReq))

	ladder.Pipe(localConn, tunnelConn)
}
