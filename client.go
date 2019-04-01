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
		u      *url.URL
		rawReq []byte
	)
	if u, err = url.Parse(self.ServerAddr); err != nil {
		return
	}
	request.HttpRequest.RequestURI = ""
	request.HttpRequest.Host = u.Host
	request.HttpRequest.Header.Add("Real-Reaquest", base64.StdEncoding.EncodeToString(request.Bytes()))
	if https {
		request.HttpRequest.Header.Add("Https", "true")
	} else {
		request.HttpRequest.Header.Add("Https", "false")
	}

	if rawReq, err = httputil.DumpRequest(request.HttpRequest, false); err != nil {
		return
	}

	// fmt.Print(string(rawReq))

	var tunnelConn net.Conn
	if tunnelConn, err = DialHttp(self.ServerAddr); err != nil {
		return
	}

	if _, err = tunnelConn.Write(rawReq); err != nil {
		return
	}

	ladder.Pipe(localConn, tunnelConn)
}
