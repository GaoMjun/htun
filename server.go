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
		hostport   string
		remoteConn net.Conn
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)

			w.Write(index)
			return
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

	// fmt.Print(string(respBytes))
	// fmt.Println("##################")

	xor(respBytes, respBytes, self.Key)
	if _, err = w.Write(respBytes); err != nil {
		return
	}
	w.(http.Flusher).Flush()

	// body
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
