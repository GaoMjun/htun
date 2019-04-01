package htun

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/url"
	"os"
	"time"
)

var MaxSerialNumber = big.NewInt(0).SetBytes(bytes.Repeat([]byte{255}, 20))

func NewAuthority() (ca *x509.Certificate, privateKey *rsa.PrivateKey, err error) {
	var (
		name         = "htun"
		organization = name
		publicKey    crypto.PublicKey
		pkixpub      []byte
		serial       *big.Int
		keyID        []byte
		hash         = sha1.New()
		raw          []byte
	)

	if privateKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
		return
	}
	publicKey = privateKey.Public()

	if pkixpub, err = x509.MarshalPKIXPublicKey(publicKey); err != nil {
		return
	}

	if _, err = hash.Write(pkixpub); err != nil {
		return
	}
	keyID = hash.Sum(nil)

	if serial, err = rand.Int(rand.Reader, MaxSerialNumber); err != nil {
		return
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{organization},
		},
		SubjectKeyId:          keyID,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		NotBefore:             time.Now().AddDate(-1, 0, 0),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		DNSNames:              []string{name},
		IsCA:                  true,
	}

	if raw, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, publicKey, privateKey); err != nil {
		return
	}

	if ca, err = x509.ParseCertificate(raw); err != nil {
		return
	}

	return
}

func Cert(hostname string, ca *x509.Certificate, pk *rsa.PrivateKey) (cert *tls.Certificate, err error) {
	var (
		raw         []byte
		tmpl, x509c *x509.Certificate
		serial      *big.Int
	)

	if serial, err = rand.Int(rand.Reader, MaxSerialNumber); err != nil {
		return
	}

	tmpl = &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: ca.Subject.Organization,
		},
		SubjectKeyId:          ca.SubjectKeyId,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		NotBefore:             time.Now().AddDate(-1, 0, 0),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		DNSNames:              []string{hostname},
	}

	if raw, err = x509.CreateCertificate(rand.Reader, tmpl, ca, pk.Public(), pk); err != nil {
		return
	}

	if x509c, err = x509.ParseCertificate(raw); err != nil {
		return
	}

	cert = &tls.Certificate{
		Certificate: [][]byte{raw, ca.Raw},
		PrivateKey:  pk,
		Leaf:        x509c,
	}
	return
}

func LoadCert(capath, pkpath string) (ca *x509.Certificate, pk *rsa.PrivateKey, err error) {
	var (
		caRaw, pkRaw []byte
	)

	if caRaw, err = ioutil.ReadFile(capath); err != nil {
		return
	}

	if pkRaw, err = ioutil.ReadFile(pkpath); err != nil {
		return
	}

	if ca, err = x509.ParseCertificate(caRaw); err != nil {
		return
	}

	if pk, err = x509.ParsePKCS1PrivateKey(pkRaw); err != nil {
		return
	}

	return
}

func GenerateCert(caname, pkname string) (ca *x509.Certificate, pk *rsa.PrivateKey, err error) {
	if ca, pk, err = NewAuthority(); err != nil {
		return
	}

	if err = ioutil.WriteFile(caname, ca.Raw, os.ModePerm); err != nil {
		return
	}
	if err = ioutil.WriteFile(pkname, x509.MarshalPKCS1PrivateKey(pk), os.ModePerm); err != nil {
		return
	}
	return
}

func DialHttp(rawurl string) (conn net.Conn, err error) {
	var (
		u              *url.URL
		hostport, port string
	)
	if u, err = url.Parse(rawurl); err != nil {
		return
	}

	port = u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	hostport = fmt.Sprintf("%s:%s", u.Hostname(), port)

	if conn, err = net.Dial("tcp", hostport); err != nil {
		return
	}

	if port == "443" {
		tlsConfig := &tls.Config{ServerName: u.Hostname()}
		conn = tls.Client(conn, tlsConfig)
	}

	return
}
