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
		err        error
		https      = false
		req        *http.Request
		reqBytes   []byte
		writen     = 0
		hostport   string
		remoteConn net.Conn
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)

			w.Write(index)
			return
		}

		log.Println(hostport, writen)
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
	hostport = getHostPort(req, https)

	req.RequestURI = ""
	req.Header.Set("Connection", "close")
	if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
		return
	}

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

	var (
		resp      *http.Response
		respBytes []byte
	)
	if resp, err = http.ReadResponse(bufio.NewReader(remoteConn), nil); err != nil {
		return
	}

	if respBytes, err = httputil.DumpResponse(resp, false); err != nil {
		return
	}

	// header
	xor(respBytes, respBytes, self.Key)
	if _, err = w.Write(respBytes); err != nil {
		return
	}
	w.(http.Flusher).Flush()

	// body
	// resp.Body

	buffer := make([]byte, 1024*32)
	n := 0
	rd := NewXorReader(remoteConn, self.Key)
	for {
		if n, err = rd.Read(buffer); err != nil {
			return
		}

		if _, err = w.Write(buffer[:n]); err != nil {
			return
		}
		w.(http.Flusher).Flush()

		writen += n
	}
}
