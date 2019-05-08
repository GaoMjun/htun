package htun

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

type Client struct {
	LocalAddr  string
	ServerAddr string
	ServerHost string
	HttpClient *http.Client
	CA         *x509.Certificate
	PK         *rsa.PrivateKey
	Key        []byte
}

func (self *Client) Run() (err error) {
	self.HttpClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				if self.ServerHost != "" {
					addr = self.ServerHost
				}
				return net.Dial(network, addr)
			},
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
	for {
		if req, err = http.ReadRequest(bufio.NewReader(localConn)); err != nil {
			return
		}

		if req.Method == http.MethodConnect {
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

		self.doRequest(localConn, req, https)
	}
}

func (self *Client) doRequest(localConn net.Conn, r *http.Request, https bool) {
	var (
		err      error
		reqBytes []byte
		url      = self.ServerAddr + r.URL.Path
		req      *http.Request
		resp     *http.Response
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	// log.Println(getHostPort(r, https))

	if reqBytes, err = httputil.DumpRequest(r, true); err != nil {
		return
	}

	enReqBytes := make([]byte, len(reqBytes))
	xor(reqBytes, enReqBytes, self.Key)
	body := bytes.NewReader(enReqBytes)

	if req, err = http.NewRequest("POST", url, body); err != nil {
		return
	}

	if ua := r.Header.Get("User-Agent"); len(ua) > 0 {
		req.Header.Set("User-Agent", ua)
	}
	req.Header.Set("Content-Type", "image/jpeg")
	req.Header.Set("Https", "false")
	if https {
		req.Header.Set("Https", "true")
	}

	if resp, err = self.HttpClient.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	io.Copy(localConn, NewXorReader(resp.Body, self.Key))
}
