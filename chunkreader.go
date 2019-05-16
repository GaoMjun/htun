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

	n, err = self.r.Read(p[:len(p)-8])
	if n > 0 {
		fmt.Fprintf(self.buffer, "%x\r\n", n)
		self.buffer.Write(p[:n])
		self.buffer.WriteString("\r\n")
	}
	if err != nil {
		self.buffer.WriteString("0\r\n\r\n")
	}

	n = copy(p, self.buffer.Bytes())
	return
}
