// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"net"
	"os"
	"sync"
)

type listener struct {
	once          sync.Once
	ln            net.Listener
	pconn         net.PacketConn
	lnaddr        net.Addr
	addr, network string
}

func (ln *listener) normalize() (err error) {
	switch ln.network {
	case "unix":
		sniffErrorAndLog(os.RemoveAll(ln.addr))
		fallthrough
	case "tcp", "tcp4", "tcp6":
		if ln.ln, err = net.Listen(ln.network, ln.addr); err != nil {
			return
		}
		ln.lnaddr = ln.ln.Addr()
	case "udp", "udp4", "udp6":
		if ln.pconn, err = net.ListenPacket(ln.network, ln.addr); err != nil {
			return
		}
		ln.lnaddr = ln.pconn.LocalAddr()
	default:
		err = ErrUnsupportedProtocol
	}
	return
}

func (ln *listener) close() {
	ln.once.Do(func() {
		if ln.ln != nil {
			sniffErrorAndLog(ln.ln.Close())
		}
		if ln.pconn != nil {
			sniffErrorAndLog(ln.pconn.Close())
		}
		if ln.network == "unix" {
			sniffErrorAndLog(os.RemoveAll(ln.addr))
		}
	})
}

func initListener(network, addr string, _ bool) (l *listener, err error) {
	l = &listener{network: network, addr: addr}
	err = l.normalize()
	return
}
