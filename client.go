package htun

import (
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
	LocalAddr   string
	ServerAddr  string
	ServerHost  string
	HttpServer  *http.Server
	HttpsServer *http.Server
	HttpClient  *http.Client
	CA          *x509.Certificate
	PK          *rsa.PrivateKey
	Key         []byte
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

	self.HttpServer = &http.Server{
		Addr:    self.LocalAddr,
		Handler: http.HandlerFunc(self.handleHttp),
	}

	self.HttpsServer = &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.HandlerFunc(self.handleHttps),
	}

	go self.HttpsServer.ListenAndServe()
	err = self.HttpServer.ListenAndServe()
	return
}

func (self *Client) handleHttp(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		bs  []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if bs, err = httputil.DumpRequest(r, true); err != nil {
		return
	}

	if r.Method == http.MethodConnect {
		var localConn net.Conn
		if localConn, _, err = w.(http.Hijacker).Hijack(); err != nil {
			return
		}

		if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
			return
		}

		tlsConfig := &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return Cert(info.ServerName, self.CA, self.PK)
			}}
		localConn = tls.Server(localConn, tlsConfig)

		if err = self.HttpsServer.Serve(NewListener(localConn)); err != nil {
			if err == ErrNotError {
				err = nil
			}
		}
		return
	}

	enbs := make([]byte, len(bs))
	xor(bs, enbs, self.Key)
	self.DoRequest(w, r, bytes.NewReader(enbs), false)
}

func (self *Client) handleHttps(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		bs  []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if bs, err = httputil.DumpRequest(r, true); err != nil {
		return
	}

	if r.Method == http.MethodConnect {
		var localConn net.Conn
		if localConn, _, err = w.(http.Hijacker).Hijack(); err != nil {
			return
		}

		if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
			return
		}

		tlsConfig := &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return Cert(info.ServerName, self.CA, self.PK)
			}}
		localConn = tls.Server(localConn, tlsConfig)

		if err = self.HttpServer.Serve(NewListener(localConn)); err != nil {
			if err == ErrNotError {
				err = nil
			}
		}
		return
	}

	enbs := make([]byte, len(bs))
	xor(bs, enbs, self.Key)
	self.DoRequest(w, r, bytes.NewReader(enbs), true)
}

func (self *Client) DoRequest(w http.ResponseWriter, r *http.Request, body io.Reader, https bool) {
	var (
		err       error
		req       *http.Request
		resp      *http.Response
		url       = self.ServerAddr + r.URL.Path
		localConn net.Conn
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	// body = NewXorReader(body, self.Key)

	if localConn, _, err = w.(http.Hijacker).Hijack(); err != nil {
		return
	}
	defer localConn.Close()

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
