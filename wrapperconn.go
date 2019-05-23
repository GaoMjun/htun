package htun

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type WrapperConn struct {
	c         net.Conn
	hostport  string
	handshake bool
}

func (self *WrapperConn) Read(b []byte) (n int, err error) {
	if self.handshake {
		fake := []byte(fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", self.hostport))
		if len(b) < len(fake) {
			err = errors.New("read buffer not enough")
			return
		}

		n = copy(b, fake)
		return
	}

	n, err = self.c.Read(b)
	return
}
func (self *WrapperConn) Write(b []byte) (n int, err error) {
	if self.handshake {
		self.handshake = false

		n = len(b)
		return
	}

	n, err = self.c.Write(b)
	return
}
func (self *WrapperConn) Close() error {
	return self.c.Close()
}
func (self *WrapperConn) LocalAddr() net.Addr {
	return self.c.LocalAddr()
}
func (self *WrapperConn) RemoteAddr() net.Addr {
	return self.c.RemoteAddr()
}
func (self *WrapperConn) SetDeadline(t time.Time) error {
	return self.c.SetDeadline(t)
}
func (self *WrapperConn) SetReadDeadline(t time.Time) error {
	return self.c.SetReadDeadline(t)
}
func (self *WrapperConn) SetWriteDeadline(t time.Time) error {
	return self.c.SetWriteDeadline(t)
}
