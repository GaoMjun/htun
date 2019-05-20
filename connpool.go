package htun

import (
	"net"
	"sync"
)

type ConnPool struct {
	conns  map[string]net.Conn
	locker *sync.RWMutex
}

func NewConnPool() (pool *ConnPool) {
	pool = &ConnPool{map[string]net.Conn{}, &sync.RWMutex{}}
	return
}

func (self *ConnPool) Add(conn net.Conn) {
	self.locker.Lock()
	defer self.locker.Unlock()

	self.conns[conn.LocalAddr().String()] = conn
}

func (self *ConnPool) Get() (conn net.Conn) {
	self.locker.Lock()
	defer self.locker.Unlock()

	for _, v := range self.conns {
		delete(self.conns, v.LocalAddr().String())

		conn = v
	}

	return
}
