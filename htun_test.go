package htun

import (
	"log"
	"net/http"
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
		// server    = Server{Addr: "127.0.0.1:8888"}
		ca, pk, _ = LoadCert("htun.cer", "htun.key")
		client    = Client{":8877", "https://htun01.herokuapp.com/", ca, pk}
	)

	// go server.Run()
	client.Run()
}
