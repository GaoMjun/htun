package htun

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"strings"
	"testing"
)

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)

	go func() {
		log.Println(http.ListenAndServe(":6061", nil))
	}()
}

func TestHtun(t *testing.T) {
	var (
		key = []byte("12345")

		ca, pk, _ = LoadCert("htun.cer", "htun.key")

		server = Server{Addr: "127.0.0.1:8888", Key: key}
		client = Client{
			LocalAddr:  ":19999",
			ServerAddr: "http://1.cdn.dnsv1.com",
			ServerHost: "180.96.32.99:80",
			CA:         ca,
			PK:         pk,
			Key:        key,
		}
	)

	go server.Run("", "")
	client.Run()
}

func TestChunked(t *testing.T) {
	var (
		err  error
		conn net.Conn
		// resp      *http.Response
		// respBytes []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if conn, err = net.Dial("tcp", "f.3.cn:443"); err != nil {
		return
	}
	defer conn.Close()

	tlsConfig := &tls.Config{ServerName: "f.3.cn"}
	conn = tls.Client(conn, tlsConfig)

	req := "GET / HTTP/1.1\r\nHost: bigbuckbunny.cf\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3\r\nAccept-Encoding: gzip, deflate\r\nAccept-Language: en-US,en;q=0.9\r\nCache-Control: max-age=0\r\nCookie: __cfduid=d7d59c7c90a76f8b112f620bfa2aeecb21557221690\r\nProxy-Connection: keep-alive\r\nUpgrade-Insecure-Requests: 1\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.131 Safari/537.36\r\n\r\n"
	req = "GET /recommend/focus_gateway/get?pin=&uuid=1553841870597468909488&jda=122270672.1553841870597468909488.1553841871.1554362740.1557281638.3&area=&callback=jsonpFocus&_=1557282084538 HTTP/1.1\r\nHost: f.3.cn\r\n\r\n"

	if _, err = conn.Write([]byte(req)); err != nil {
		return
	}

	var (
		buffer = make([]byte, 1024)
		n      = 0
	)
	for {
		if n, err = conn.Read(buffer); err != nil {
			fmt.Println(string(buffer[:n]))

		}

		fmt.Println(string(buffer[:n]))
	}

	// if resp, err = http.ReadResponse(bufio.NewReader(conn), nil); err != nil {
	// 	return
	// }

	// if respBytes, err = httputil.DumpResponse(resp, true); err != nil {
	// 	return
	// }

	// fmt.Println(string(respBytes))

}

func TestMIMT(t *testing.T) {
	var (
		err        error
		l          net.Listener
		ca, pk, _  = LoadCert("htun.cer", "htun.key")
		handleConn func(conn net.Conn, https bool)
		doRequest  func(conn net.Conn, req *http.Request, https bool)
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if l, err = net.Listen("tcp", "0.0.0.0:19999"); err != nil {
		return
	}

	handleConn = func(localConn net.Conn, https bool) {
		defer localConn.Close()

		var (
			err error
			req *http.Request
		)
		defer func() {
			if err != nil && err != io.EOF {
				log.Println(err)
			}
		}()
		for {
			if req, err = http.ReadRequest(bufio.NewReader(localConn)); err != nil {
				return
			}

			if req.Method == http.MethodConnect {
				if _, err = fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
					return
				}

				tlsConfig := &tls.Config{
					GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
						return Cert(info.ServerName, ca, pk)
					}}
				localConn = tls.Server(localConn, tlsConfig)
				handleConn(localConn, true)
				return
			}

			doRequest(localConn, req, https)
		}
	}

	doRequest = func(localConn net.Conn, req *http.Request, https bool) {
		var (
			err        error
			reqBytes   []byte
			hostport   = getHostPort(req, https)
			remoteConn net.Conn
		)
		defer func() {
			if err != nil {
				log.Println(err)
			}
		}()

		log.Println(hostport)

		req.RequestURI = ""
		req.Header.Set("Connection", "close")
		if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
			return
		}

		if remoteConn, err = net.Dial("tcp", hostport); err != nil {
			err = errors.New(fmt.Sprint("dial failed", hostport))
			return
		}
		defer remoteConn.Close()

		if https {
			remoteConn = tls.Client(remoteConn, &tls.Config{ServerName: strings.Split(req.Host, ":")[0]})
		}

		if _, err = remoteConn.Write(reqBytes); err != nil {
			return
		}

		io.Copy(localConn, remoteConn)
	}

	for {
		if conn, err := l.Accept(); err == nil {
			go handleConn(conn, false)
			continue
		}
	}
}
