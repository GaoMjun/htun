package htun

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/GaoMjun/ladder"
)

type Client struct {
	LocalAddr  string
	ServerAddr string
	ServerHost string
	HttpClient *http.Client
	CA         *x509.Certificate
	PK         *rsa.PrivateKey
	Key        []byte

	connPool *ConnPool
}

func ClientRun(args []string) (err error) {
	flags := flag.NewFlagSet("client", flag.PanicOnError)
	addr := flags.String("l", ":80", "server listen address")
	pass := flags.String("k", "", "password")
	sa := flags.String("sa", "", "server http address")
	sh := flags.String("sh", "", "server http host")
	capath := flags.String("ca", "", "certificate file")
	pkpath := flags.String("pk", "", "private key file")
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
		connPool:   NewConnPool(),
	}
	err = client.Run()

	return
}

func (self *Client) Run() (err error) {
	self.HttpClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (conn net.Conn, err error) {
				if conn = self.connPool.Get(); conn != nil {
					return
				}

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

	self.doRequest(localConn, req, https)
}

func (self *Client) doRequest(localConn net.Conn, r *http.Request, https bool) {
	var (
		err      error
		reqBytes []byte
		url      = self.ServerAddr + "/" + hex.EncodeToString([]byte(r.URL.Path))
		req      *http.Request
		resp     *http.Response
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	log.Println(getHostPort(r, https))

	if reqBytes, err = httputil.DumpRequest(r, true); err != nil {
		return
	}

	enReqBytes := make([]byte, len(reqBytes))
	xor(reqBytes, enReqBytes, self.Key)
	body := bytes.NewReader(enReqBytes)

	if req, err = http.NewRequest("POST", url, body); err != nil {
		return
	}

	token, _ := ladder.GenerateToken(string(self.Key), string(self.Key))
	req.Header.Set("Token", token)
	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Https", fmt.Sprintf("%v", https))

	if resp, err = self.HttpClient.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	buffer := make([]byte, 1024*32)
	n := 0
	rd := NewXorReader(resp.Body, self.Key)
	for {
		n, err = rd.Read(buffer)
		if n > 0 {
			if _, err = localConn.Write(buffer[:n]); err != nil {
				return
			}
		}

		if err != nil {
			return
		}
	}
}
