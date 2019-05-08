package htun

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Server struct {
	Addr string
}

func (self *Server) Run(capath, pkpath string) (err error) {
	err = http.ListenAndServe(self.Addr, http.HandlerFunc(handleHttp))
	return
}

func handleHttp(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		https    = r.Header.Get("Https")
		req      *http.Request
		reqBytes []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if req, err = http.ReadRequest(bufio.NewReader(r.Body)); err != nil {
		return
	}
	req.RequestURI = ""
	req.Header.Set("Connection", "close")
	if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
		return
	}

	var (
		hostandport = strings.Split(req.Host, ":")
		host        = hostandport[0]
		port        = "80"
		hostport    string
		remoteConn  net.Conn
	)
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

	if _, err = remoteConn.Write(reqBytes); err != nil {
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	io.Copy(w, remoteConn)
}
