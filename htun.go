package htun

import (
	"bufio"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

func Process(src, dst net.Conn, key []byte, host string, https bool, ca *x509.Certificate, pk *rsa.PrivateKey) {
	var (
		err      error
		req      *http.Request
		reqBytes []byte
		resp     *http.Response
	)
	defer func() {
		src.Close()
		dst.Close()

		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		for {
			if req, err = http.ReadRequest(bufio.NewReader(src)); err != nil {
				return
			}

			if req.Method == http.MethodConnect {
				if _, err = fmt.Fprint(src, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
					return
				}

				if req.URL.Port() == "80" {
					Process(src, dst, key, host, false, ca, pk)
					return
				}

				tlsConfig := &tls.Config{
					GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
						return Cert(info.ServerName, ca, pk)
					}}
				src = tls.Server(src, tlsConfig)
				Process(src, dst, key, host, true, ca, pk)
				return
			}

			if reqBytes, err = httputil.DumpRequest(req, true); err != nil {
				return
			}

			xor(reqBytes, reqBytes, key)

			if _, err = fmt.Fprintf(dst, "POST /%s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nContent-Type: application/octet-stream\r\nHttps: %v\r\nContent-Length: %d\r\n\r\n",
				hex.EncodeToString([]byte(req.URL.Path)), host, req.Header.Get("User-Agent"), https, len(reqBytes)); err != nil {
				return
			}

			if _, err = dst.Write(reqBytes); err != nil {
				return
			}
		}
	}()

	for {
		if resp, err = http.ReadResponse(bufio.NewReader(dst), nil); err != nil {
			return
		}

		io.Copy(src, NewXorReader(resp.Body, key))
	}
}
