package htun

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/golang/snappy"

	"github.com/GaoMjun/ladder"
)

type Server struct {
	Addr string
}

func (self *Server) Run(capath, pkpath string) (err error) {
	var (
		l         net.Listener
		tlsConfig *tls.Config
	)
	if l, err = net.Listen("tcp", self.Addr); err != nil {
		return
	}
	defer l.Close()

	log.Println("server run at", l.Addr())

	if l.Addr().(*net.TCPAddr).Port == 443 {
		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(capath, pkpath); err != nil {
			return
		}

		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	for {
		conn, _ := l.Accept()
		if tlsConfig != nil {
			conn = tls.Server(conn, tlsConfig)
		}

		go self.handleConn(conn)
	}

	return
}

func (self *Server) handleConn(tunnelConn net.Conn) {
	var (
		err     error
		request *ladder.Request
	)
	defer func() {
		tunnelConn.Close()

		if err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	if request, err = ladder.NewRequest(tunnelConn); err != nil {
		return
	}

	if request.HttpRequest.Method == http.MethodConnect {
		return
	}

	// fmt.Print(request.Dump())

	var (
		s           string
		https       string
		r           = request.HttpRequest
		realRequest []byte
		req         *http.Request
	)
	defer func() {
		fmt.Fprint(tunnelConn, "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n")
		tunnelConn.Write(index)
	}()

	if s = r.Header.Get("Real-Reaquest"); len(s) <= 0 {
		return
	}
	https = r.Header.Get("Https")

	if realRequest, err = base64.StdEncoding.DecodeString(s); err != nil {
		return
	}

	if realRequest, err = snappy.Decode(nil, realRequest); err != nil {
		return
	}

	if req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(realRequest))); err != nil {
		return
	}

	var (
		port        string
		hostport    = req.Host
		hostandport = strings.Split(hostport, ":")
		host        = hostandport[0]
		remoteConn  net.Conn
	)
	port = "80"
	if https == "true" {
		port = "443"
	}
	if len(hostandport) == 2 {
		port = hostandport[1]
	}
	hostport = fmt.Sprintf("%s:%s", host, port)
	log.Println(hostport)

	if remoteConn, err = net.Dial("tcp", hostport); err != nil {
		return
	}
	defer remoteConn.Close()

	if https == "true" {
		tlsConfig := &tls.Config{ServerName: host}
		remoteConn = tls.Client(remoteConn, tlsConfig)
	}

	if _, err = remoteConn.Write(realRequest); err != nil {
		return
	}

	ladder.Pipe(tunnelConn, remoteConn)
}
