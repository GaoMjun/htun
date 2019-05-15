package htun

import (
	"bytes"
	"fmt"
	"io"
)

type ChunkReader struct {
	r      io.Reader
	buffer *bytes.Buffer
}

func NewChunkReader(r io.Reader) (cr *ChunkReader) {
	cr = &ChunkReader{}
	cr.r = r
	cr.buffer = &bytes.Buffer{}
	return
}

func (self *ChunkReader) Read(p []byte) (n int, err error) {
	self.buffer.Reset()

	if n, err = self.r.Read(p[:len(p)-8]); err != nil && n <= 0 {
		n = copy(p, []byte("0\r\n"))
		return
	}
	err = nil

	fmt.Fprintf(self.buffer, "%x\r\n", n)
	self.buffer.Write(p[:n])
	self.buffer.WriteString("\r\n")

	n = copy(p, self.buffer.Bytes())
	return
}
