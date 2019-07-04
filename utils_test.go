package htun

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
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

func TestHTTPSRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://baidu.com/a?b=c", nil)
	req2, _ := http.NewRequest("GET", "http://baidu.com/a?b=c", nil)

	log.Println(req.URL)
	log.Println(req2.URL)

	bs, _ := httputil.DumpRequest(req, true)
	bs2, _ := httputil.DumpRequest(req2, true)

	fmt.Print(string(bs))
	fmt.Print(string(bs2))
}

func TestRequest(t *testing.T) {
	var (
		err error
		req *http.Request
		// reqBytes  []byte
		resp      *http.Response
		respBytes []byte
		body      []byte

		httpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        0,
				MaxIdleConnsPerHost: 128,
				MaxConnsPerHost:     0,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if req, err = http.NewRequest("GET", "http://fancy-hat-9719.bigbuckbunny.workers.dev/", nil); err != nil {
		return
	}

	req.Header.Add("xprotocol", "http")
	req.Header.Add("xhost", "google.com")

	// if reqBytes, err = httputil.DumpRequest(req, false); err != nil {
	// 	return
	// }
	// fmt.Print(string(reqBytes))

	if resp, err = httpClient.Do(req); err != nil {
		return
	}

	resp.TransferEncoding = nil
	if respBytes, err = httputil.DumpResponse(resp, false); err != nil {
		return
	}

	fmt.Print(string(respBytes))

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}
	fmt.Print(string(body))
}
