package htun

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Server struct {
	Addr string
	Key  []byte
}

func (self *Server) Run(capath, pkpath string) (err error) {
	err = http.ListenAndServe(self.Addr, http.HandlerFunc(self.handleHttp))
	return
}

func (self *Server) handleHttp(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		https    = false
		req      *http.Request
		reqBytes []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)

			w.Write(index)
		}
	}()

	if r.Header.Get("Https") == "true" {
		https = true
	} else if r.Header.Get("Https") == "false" {
		https = false
	} else {
		err = errors.New("no [Https] header")
		return
	}

	if req, err = http.ReadRequest(bufio.NewReader(NewXorReader(r.Body, self.Key))); err != nil {
		return
	}
	req.RequestURI = ""
	req.Header.Set("Connection", "close")
	if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
		return
	}

	var (
		hostport   = getHostPort(req, https)
		remoteConn net.Conn
	)

	log.Println(hostport)
	// fmt.Println(string(reqBytes))
	// fmt.Println("##################")

	if remoteConn, err = net.Dial("tcp", hostport); err != nil {
		return
	}
	defer remoteConn.Close()

	if https {
		tlsConfig := &tls.Config{ServerName: strings.Split(req.Host, ":")[0]}
		remoteConn = tls.Client(remoteConn, tlsConfig)
	}

	if _, err = remoteConn.Write(reqBytes); err != nil {
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	io.CopyBuffer(w, NewXorReader(remoteConn, self.Key), make([]byte, 1024))
}
