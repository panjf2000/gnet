// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build !darwin,!netbsd,!freebsd,!openbsd,!dragonfly,!linux

package gnet

import (
	"errors"
	"net"
	"os"
)

func (ln *listener) close() {
	if ln.ln != nil {
		ln.ln.Close()
	}
	if ln.pconn != nil {
		ln.pconn.Close()
	}
	if ln.network == "unix" {
		os.RemoveAll(ln.addr)
	}
}

func (ln *listener) system() error {
	return nil
}

func serve(eventHandler EventHandler, listeners []*listener) error {
	return errors.New("Unsupported platform in gnet")
}

func ReusePortListenPacket(proto, addr string) (net.PacketConn, error) {
	return nil, errors.New("reuseport is not available")
}

func ReusePortListen(proto, addr string) (net.Listener, error) {
	return nil, errors.New("reuseport is not available")
}
