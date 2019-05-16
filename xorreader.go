package htun

import (
	"io"
)

type XorReader struct {
	r    io.Reader
	k    []byte
	rbuf []byte
}

func NewXorReader(r io.Reader, key []byte) (xer *XorReader) {
	xer = &XorReader{}
	xer.r = r
	xer.k = key
	return
}

func (self *XorReader) Read(p []byte) (n int, err error) {
	n, err = self.r.Read(p)
	if n > 0 {
		if n > len(self.rbuf) {
			self.rbuf = make([]byte, n)
		}
		xor(p[:n], self.rbuf, self.k)
		copy(p[:n], self.rbuf)
	}

	return
}
