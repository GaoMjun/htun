package main

import "C"
import (
	"strings"

	"github.com/GaoMjun/htun"
)

func main() {}

//export htunclient_run
func htunclient_run(sa, sh string) {
	if !strings.Contains(sh, ":") {
		sh = sh + ":80"
	}

	go htun.ClientRun([]string{
		"-l", "127.0.0.1:1999",
		"-k", "12345",
		"-sa", sa,
		"-sh", sh})
}
