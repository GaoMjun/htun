package main

import (
	"errors"
	"log"
	"os"

	"github.com/GaoMjun/htun"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func main() {
	var (
		err  error
		mode string
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if len(os.Args) < 2 {
		err = errors.New("invalid arguments")
		return
	}

	mode = os.Args[1]

	if mode != "server" && mode != "client" {
		err = errors.New("invalid arguments")
		return
	}

	if mode == "server" {
		err = htun.ServerRun(os.Args[2:])
		return
	}

	if mode == "client" {
		err = htun.ClientRun(os.Args[2:])
		return
	}
}
