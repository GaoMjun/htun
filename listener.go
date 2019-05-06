package htun

import (
	"errors"
	"log"
	"net"
	"time"
)

type Listener struct {
	conn    *WrapConn
	closeCh chan bool
}

func NewListener(conn net.Conn) (listener *Listener) {
	listener = &Listener{}
	listener.closeCh = make(chan bool)
	listener.conn = &WrapConn{conn, func() {
		listener.closeCh <- true
	}}
	return
}

func (self *Listener) Accept() (conn net.Conn, err error) {
	if self.conn != nil {
		conn = self.conn
		self.conn = nil
		return
	}
	log.Println("wait...")
	<-self.closeCh
	err = errors.New("not error")
	log.Println("end...")
	return
}

func (self *Listener) Close() (err error) {
	log.Println("close...")
	self.closeCh <- true
	return
}

func (self *Listener) Addr() (addr net.Addr) {
	return
}

type WrapConn struct {
	conn net.Conn
	f    func()
}

func (self *WrapConn) Read(b []byte) (n int, err error) {
	if n, err = self.conn.Read(b); err != nil {
		if self.f != nil {
			self.f()
			self.f = nil
		}
	}
	return
}

func (self *WrapConn) Write(b []byte) (n int, err error) {
	if n, err = self.conn.Write(b); err != nil {
		if self.f != nil {
			self.f()
			self.f = nil
		}
	}
	return
}

func (self *WrapConn) Close() error {
	if self.f != nil {
		self.f()
		self.f = nil
	}
	return self.conn.Close()
}

func (self *WrapConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
}

func (self *WrapConn) RemoteAddr() net.Addr {
	return self.conn.RemoteAddr()
}

func (self *WrapConn) SetDeadline(t time.Time) error {
	return self.conn.SetDeadline(t)
}

func (self *WrapConn) SetReadDeadline(t time.Time) error {
	return self.conn.SetReadDeadline(t)
}

func (self *WrapConn) SetWriteDeadline(t time.Time) error {
	return self.conn.SetWriteDeadline(t)
}
