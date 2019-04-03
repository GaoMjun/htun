package htun

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
)

func TestCA(t *testing.T) {
	var (
		err error
		ca  *x509.Certificate
		pk  *rsa.PrivateKey
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if ca, pk, err = NewAuthority(); err != nil {
		return
	}

	log.Println(ca, pk)

	ioutil.WriteFile("htun.cer", ca.Raw, os.ModePerm)
	ioutil.WriteFile("htun.key", x509.MarshalPKCS1PrivateKey(pk), os.ModePerm)
}

func TestCertLoad(t *testing.T) {
	var (
		err          error
		caRaw, pkRaw []byte
		ca           *x509.Certificate
		pk           *rsa.PrivateKey
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if caRaw, err = ioutil.ReadFile("htun.cer"); err != nil {
		return
	}

	if pkRaw, err = ioutil.ReadFile("htun.key"); err != nil {
		return
	}

	if ca, err = x509.ParseCertificate(caRaw); err != nil {
		return
	}

	if pk, err = x509.ParsePKCS1PrivateKey(pkRaw); err != nil {
		return
	}

	log.Println(ca, pk)
}

func TestDialHttp(t *testing.T) {
	DialHttp("https://baidu.com/", "")
}

func TestHTTPCONNECT(t *testing.T) {
	var (
		err    error
		conn   net.Conn
		buffer = make([]byte, 4096)
		n      = 0
	)

	if conn, err = net.Dial("tcp", "baidu.com:80"); err != nil {
		return
	}

	fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: baidu.com\r\n\r\n")

	if n, err = conn.Read(buffer); err != nil {
		return
	}

	fmt.Println(string(buffer[:n]))
}
