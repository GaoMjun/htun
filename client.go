package htun

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
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
	LocalAddr   string
	ServerAddr  string
	ServerHost  string
	HttpServer  *http.Server
	HttpsServer *http.Server
	HttpClient  *http.Client
	CA          *x509.Certificate
	PK          *rsa.PrivateKey
}

func (self *Client) Run() (err error) {
	// var l net.Listener
	// if l, err = net.Listen("tcp", self.LocalAddr); err != nil {
	// 	return
	// }
	// defer l.Close()

	// log.Println("client run at", l.Addr())

	// for {
	// 	conn, _ := l.Accept()

	// 	go self.handleConn(conn, false)
	// }

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

	// fmt.Println(string(bs))

	self.DoRequest(w, r, bytes.NewReader(bs), false)
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

	// fmt.Println(string(bs))

	self.DoRequest(w, r, bytes.NewReader(bs), true)
}

func (self *Client) DoRequest(w http.ResponseWriter, r *http.Request, body io.Reader, https bool) {
	var (
		err         error
		req         *http.Request
		resp, resp2 *http.Response
		url         = self.ServerAddr + r.URL.Path
		localConn   net.Conn
		resp2Bytes  []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if localConn, _, err = w.(http.Hijacker).Hijack(); err != nil {
		return
	}

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

	if resp2, err = http.ReadResponse(bufio.NewReader(resp.Body), nil); err != nil {
		return
	}

	if resp2Bytes, err = httputil.DumpResponse(resp2, false); err != nil {
		return
	}

	// fmt.Println(string(resp2Bytes))

	if _, err = localConn.Write(resp2Bytes); err != nil {
		return
	}

	_, err = io.Copy(localConn, resp.Body)
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
		ip         = ""
	)
	if u, err = url.Parse(self.ServerAddr); err != nil {
		return
	}
	request.HttpRequest.Host = u.Host
	request.HttpRequest.RequestURI = "/" + hex.EncodeToString([]byte(request.HttpRequest.RequestURI))
	request.HttpRequest.Header.Del("Origin")
	request.HttpRequest.Header.Del("Referer")
	request.HttpRequest.Header.Add("Real-Reaquest", hex.EncodeToString(snappy.Encode(nil, request.Bytes())))
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
