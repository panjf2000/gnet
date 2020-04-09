// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"net"
	"os"
	"sync"
)

type listener struct {
	ln            net.Listener
	once          sync.Once
	pconn         net.PacketConn
	lnaddr        net.Addr
	addr, network string
}

func (ln *listener) system() error {
	return nil
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
