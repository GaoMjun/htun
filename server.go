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

	"github.com/GaoMjun/ladder"
)

type Server struct {
	Addr string
}

func (self *Server) Run() (err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", self.Addr); err != nil {
		return
	}
	log.Println("server run at", l.Addr())

	for {
		conn, _ := l.Accept()

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
		s     string
		https string
		r     = request.HttpRequest
	)
	if s = r.Header.Get("Real-Reaquest"); len(s) <= 0 {
		return
	}
	if https = r.Header.Get("Https"); len(s) <= 0 {
		return
	}

	var realRequest []byte
	if realRequest, err = base64.StdEncoding.DecodeString(s); err != nil {
		return
	}

	var req *http.Request
	if req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(realRequest))); err != nil {
		return
	}

	var (
		port        string
		hostport    = req.Host
		hostandport = strings.Split(hostport, ":")
		host        = hostandport[0]
	)
	if len(hostandport) == 2 {
		port = hostandport[1]
	} else {
		if https == "true" {
			port = "443"
		} else {
			port = "80"
		}
	}
	hostport = fmt.Sprintf("%s:%s", host, port)
	log.Println(hostport)

	var remoteConn net.Conn
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
