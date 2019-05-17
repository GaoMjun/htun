package htun

import (
	"bufio"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/GaoMjun/ladder"
)

type Server struct {
	Addr string
	Key  []byte
}

func ServerRun(args []string) (err error) {
	flags := flag.NewFlagSet("server", flag.PanicOnError)
	addr := flags.String("l", ":80", "server listen address")
	pass := flags.String("k", "", "password")
	flags.Parse(args)

	server := Server{Addr: *addr, Key: []byte(*pass)}
	err = server.Run()

	return
}

func (self *Server) Run() (err error) {
	err = http.ListenAndServe(self.Addr, http.HandlerFunc(self.handleHttp))
	return
}

func (self *Server) handleHttp(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		https      = false
		req        *http.Request
		reqBytes   []byte
		hostport   string
		remoteConn net.Conn
		token      = r.Header.Get("Token")
		tokenOk    bool
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)

			w.Write(index)
			return
		}
	}()

	if len(token) <= 0 {
		err = errors.New("token invalid, no token")
		return
	}

	if tokenOk, err = ladder.CheckToken(string(self.Key), string(self.Key), token); err != nil {
		return
	}

	if tokenOk != true {
		err = errors.New(fmt.Sprint("token invalid,", token))
		return
	}

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
	log.Println(hostport)

	req.RequestURI = ""
	req.Header.Set("Connection", "close")
	if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
		return
	}

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
		chunked   bool
	)
	if resp, err = http.ReadResponse(bufio.NewReader(remoteConn), nil); err != nil {
		return
	}

	for _, v := range resp.TransferEncoding {
		if v == "chunked" {
			chunked = true
			break
		}
	}

	if respBytes, err = httputil.DumpResponse(resp, false); err != nil {
		return
	}

	xor(respBytes, respBytes, self.Key)
	if _, err = w.Write(respBytes); err != nil {
		return
	}
	w.(http.Flusher).Flush()

	var reader io.Reader = NewXorReader(resp.Body, self.Key)
	if chunked {
		reader = NewXorReader(NewChunkReader(resp.Body), self.Key)
	}

	buffer := make([]byte, 1024*32)
	n := 0
	for {
		n, err = reader.Read(buffer)
		if n > 0 {
			if _, err2 := w.Write(buffer[:n]); err2 != nil {
				err = err2
				return
			}
			w.(http.Flusher).Flush()
		}

		if err != nil {
			return
		}
	}
}
