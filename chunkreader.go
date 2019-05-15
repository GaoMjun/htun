package htun

import (
	"io"
)

type ChunkReader struct {
	r io.Reader
}

func NewChunkReader() {

}

func (self *ChunkReader) Read(p []byte) (n int, err error) {
	t := make([]byte, len(p)-8)
	if n, err = self.r.Read(t); err != nil && n <= 0 {
		return
	}

	// byte

	err = nil
	return
}
