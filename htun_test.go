package htun

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"testing"
)

func init() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	go func() {
		log.Println(http.ListenAndServe(":6061", nil))
	}()
}

func TestHtun(t *testing.T) {
	var (
		server    = Server{Addr: "127.0.0.1:8888"}
		ca, pk, _ = LoadCert("htun.cer", "htun.key")
		client    = Client{
			LocalAddr:  ":19999",
			ServerAddr: "http://test.ceewa.com",
			ServerHost: "127.0.0.1:8888",
			CA:         ca,
			PK:         pk,
		}
	)

	go server.Run("", "")
	client.Run()
}

func TestChunked(t *testing.T) {
	var (
		err       error
		conn      net.Conn
		resp      *http.Response
		respBytes []byte
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if conn, err = net.Dial("tcp", "www.baidu.com:80"); err != nil {
		return
	}
	defer conn.Close()

	req := "GET / HTTP/1.1\r\nHost: bigbuckbunny.cf\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3\r\nAccept-Encoding: gzip, deflate\r\nAccept-Language: en-US,en;q=0.9\r\nCache-Control: max-age=0\r\nCookie: __cfduid=d7d59c7c90a76f8b112f620bfa2aeecb21557221690\r\nProxy-Connection: keep-alive\r\nUpgrade-Insecure-Requests: 1\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.131 Safari/537.36\r\n\r\n"
	req = "GET / HTTP/1.1\r\nHost: www.baidu.com\r\n\r\n"

	if _, err = conn.Write([]byte(req)); err != nil {
		return
	}

	if resp, err = http.ReadResponse(bufio.NewReader(conn), nil); err != nil {
		return
	}

	if respBytes, err = httputil.DumpResponse(resp, false); err != nil {
		return
	}

	fmt.Println(string(respBytes))

}
