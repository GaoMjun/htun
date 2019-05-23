package htun

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

func socksServerRun(addr string, handle func(net.Conn, bool)) {
	var (
		err        error
		l          net.Listener
		handleConn = func(conn net.Conn) {
			defer conn.Close()

			var (
				err      error
				hostport string
			)
			defer func() {
				if err != nil {
					log.Println(err)
				}
			}()

			//
			buf := make([]byte, 2)
			if _, err = io.ReadFull(conn, buf); err != nil {
				return
			}
			if buf[0] != 5 {
				err = errors.New("not support")
				return
			}
			methodcount := int(buf[1])
			if !(methodcount >= 1 && methodcount <= 255) {
				err = errors.New("not support")
				return
			}
			buf = make([]byte, methodcount)
			if _, err = io.ReadFull(conn, buf); err != nil {
				return
			}
			noauth := false
			for i := 0; i < methodcount; i++ {
				if buf[i] == 0 {
					noauth = true
					break
				}
			}
			if noauth == false {
				err = errors.New("not support")
				return
			}
			buf = make([]byte, 2)
			buf[0] = 5
			buf[1] = 0
			if _, err = conn.Write(buf); err != nil {
				return
			}

			//
			buf = make([]byte, 4)
			if _, err = io.ReadFull(conn, buf); err != nil {
				return
			}
			if !(buf[0] == 5 && buf[1] == 1 && buf[2] == 0) {
				err = errors.New("not support")
				return
			}
			len := 0
			atyp := int(buf[3])
			if atyp == 1 {
				len = 4 + 2
			} else if atyp == 4 {
				len = 16 + 2
			} else if atyp == 3 {
				buf = make([]byte, 1)
				if _, err = io.ReadFull(conn, buf); err != nil {
					return
				}
				len = int(buf[0]) + 2
			} else {
				err = errors.New("not support")
				return
			}
			buf = make([]byte, len)
			if _, err = io.ReadFull(conn, buf); err != nil {
				return
			}
			port := int(buf[len-2])<<8 | int(buf[len-1])
			if atyp == 1 {
				hostport = fmt.Sprintf("%s:%d", net.IP(buf[:net.IPv4len]).String(), port)
			} else if atyp == 4 {
				hostport = fmt.Sprintf("%s:%d", net.IP(buf[:net.IPv6len]).String(), port)
			} else if atyp == 3 {
				hostport = fmt.Sprintf("%s:%d", string(buf[:len-3]), port)
			}
			buf = make([]byte, 10)
			buf[0] = 5
			buf[1] = 0
			buf[2] = 0
			buf[3] = 1
			if _, err = conn.Write(buf); err != nil {
				return
			}

			//
			handle(&WrapperConn{conn, hostport, true}, false)
		}
	)
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	for {
		if conn, _err := l.Accept(); _err == nil {
			go handleConn(conn)
		}
	}
}
