package htun

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/GaoMjun/ladder/httpstream"

	"github.com/GaoMjun/ladder"
)

type Server struct {
	Addr       string
	Key        []byte
	HttpClient *http.Client
}

func ServerRun(args []string) (err error) {
	flags := flag.NewFlagSet("server", flag.PanicOnError)
	addr := flags.String("l", ":80", "server listen address")
	pass := flags.String("k", "", "password")
	flags.Parse(args)

	server := Server{
		Addr: *addr,
		Key:  []byte(*pass),
		HttpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        0,
				MaxIdleConnsPerHost: 128,
				MaxConnsPerHost:     0,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	err = server.Run()

	return
}

func (self *Server) Run() (err error) {
	upgrader := httpstream.NewUpgrader()
	go func() {
		for {
			stream := upgrader.Accept()
			go self.handleStream(stream)
		}
	}()

	err = http.ListenAndServe(self.Addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			tokenOk    bool
			token      = r.Header.Get("HTTPStream-Token")
			streamid   = r.Header.Get("HTTPStream-Key")
			remoteHost = r.Header.Get("HTTPStream-Host")
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

		if len(streamid) <= 0 {
			err = errors.New("no streamid")
			return
		}

		if len(remoteHost) <= 0 {
			err = errors.New("no remote host")
			return
		}

		upgrader.Upgrade(w, r)
	}))
	return
}

func (self *Server) handleStream(tunnelConn *httpstream.Conn) {
	defer tunnelConn.Close()

	var (
		err        error
		remoteConn net.Conn
	)
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	log.Println(tunnelConn.RemoteHost)
	// return

	if remoteConn, err = net.Dial("tcp", tunnelConn.RemoteHost); err != nil {
		return
	}

	ladder.Pipe(ladder.NewConnWithXor(tunnelConn, self.Key), remoteConn)
}
