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
	Encryption bool
}

func ServerRun(args []string) (err error) {
	flags := flag.NewFlagSet("server", flag.PanicOnError)
	addr := flags.String("l", ":80", "server listen address")
	pass := flags.String("k", "", "password")
	m := flags.Bool("m", true, "encrypt data")
	flags.Parse(args)

	server := Server{
		Addr:       *addr,
		Key:        []byte(*pass),
		Encryption: *m,
	}
	err = server.Run()

	return
}

func (self *Server) Run() (err error) {
	upgrader := httpstream.NewUpgrader()
	go func() {
		for {
			stream := upgrader.Accept()
			go self.handleStream(stream, stream.RemoteHost)
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

func (self *Server) handleStream(tunnelConn net.Conn, remoteHost string) {
	defer tunnelConn.Close()

	var (
		err        error
		remoteConn net.Conn
		host       string
		port       int
		raw        []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	if self.Encryption {
		tunnelConn = ladder.NewConnWithXor(tunnelConn, self.Key)
	}

	if host, port, raw, _, err = ladder.ParseHttpHost(tunnelConn); err != nil {
		log.Println(err)
		err = nil
	}

	if len(host) > 0 {
		remoteHost = fmt.Sprintf("%s:%d", host, port)
	}

	log.Println(remoteHost)

	if remoteConn, err = net.Dial("tcp", remoteHost); err != nil {
		return
	}
	defer remoteConn.Close()

	if len(host) > 0 {
		if _, err = remoteConn.Write(raw); err != nil {
			return
		}
	}

	ladder.PipeIoCopy(tunnelConn, remoteConn)
}
