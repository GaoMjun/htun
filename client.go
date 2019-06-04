package htun

import (
	"bufio"
	"encoding/base64"
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

type Client struct {
	LocalAddr  string
	ServerAddr string
	ServerHost string
	Key        []byte
}

func ClientRun(args []string) (err error) {
	flags := flag.NewFlagSet("client", flag.PanicOnError)
	addr := flags.String("l", ":80", "http server listen address")
	socksaddr := flags.String("sl", "", "socks5 server listen address")
	pass := flags.String("k", "", "password")
	sa := flags.String("sa", "", "server http address")
	sh := flags.String("sh", "", "server http host")
	flags.Parse(args)

	client := Client{
		LocalAddr:  *addr,
		ServerAddr: *sa,
		ServerHost: *sh,
		Key:        []byte(*pass),
	}

	if len(*socksaddr) > 0 {
		go socksServerRun(*socksaddr, client.handleConn)
	}

	err = client.Run()

	return
}

func (self *Client) Run() (err error) {
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	var l net.Listener
	if l, err = net.Listen("tcp", self.LocalAddr); err != nil {
		return
	}

	for {
		if conn, err := l.Accept(); err == nil {
			go self.handleConn(conn)
			continue
		}
	}
}

func (self *Client) handleConn(localConn net.Conn) {
	defer localConn.Close()

	var (
		err        error
		req        *http.Request
		tunnelConn *httpstream.Conn
		token, _   = ladder.GenerateToken(string(self.Key), string(self.Key))
	)
	defer func() {
		if err != nil && err != io.EOF {
			log.Println(err)
		}
	}()

	if req, err = http.ReadRequest(bufio.NewReader(localConn)); err != nil {
		return
	}

	if req.Method != http.MethodConnect {
		err = errors.New("only support connect")
		return
	}

	if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		return
	}

	log.Println(req.Host)

	header := http.Header{}
	header.Set("HTTPStream-Host", base64.StdEncoding.EncodeToString([]byte(req.Host)))
	header.Set("HTTPStream-Token", token)

	if tunnelConn, err = httpstream.Dial(self.ServerAddr, self.ServerHost, header); err != nil {
		return
	}
	defer tunnelConn.Close()

	ladder.Pipe(localConn, ladder.NewConnWithXor(tunnelConn, self.Key))
}
