package main

import (
	"log"
	"os"

	"github.com/GaoMjun/htun"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func main() {
	server := htun.Server{Addr: os.Args[1], Key: []byte(os.Args[2])}
	server.Run("", "")
}
