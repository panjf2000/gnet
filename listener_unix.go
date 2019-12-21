// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func (ln *listener) close() {
	ln.once.Do(
		func() {
			if ln.f != nil {
				sniffError(ln.f.Close())
			}
			if ln.ln != nil {
				sniffError(ln.ln.Close())
			}
			if ln.pconn != nil {
				sniffError(ln.pconn.Close())
			}
			if ln.network == "unix" {
				sniffError(os.RemoveAll(ln.addr))
			}
		})
}

// system takes the net listener and detaches it from it's parent
// event loop, grabs the file descriptor, and makes it non-blocking.
func (ln *listener) system() error {
	var err error
	switch netln := ln.ln.(type) {
	case nil:
		switch pconn := ln.pconn.(type) {
		case *net.UDPConn:
			ln.f, err = pconn.File()
		}
	case *net.TCPListener:
		ln.f, err = netln.File()
	case *net.UnixListener:
		ln.f, err = netln.File()
	}
	if err != nil {
		ln.close()
		return err
	}
	ln.fd = int(ln.f.Fd())
	return unix.SetNonblock(ln.fd, true)
}
