// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build !darwin,!netbsd,!freebsd,!openbsd,!dragonfly,!linux,!windows

package gnet

import "net"

type server struct {
	subEventLoopSet loadBalancer // event-loops for handling events
}

type eventloop struct {
	connCount int32 // number of active connections in event-loop
}

type listener struct {
	ln            net.Listener
	pconn         net.PacketConn
	lnaddr        net.Addr
	addr, network string
}

func (ln *listener) renormalize() error {
	return nil
}

func (ln *listener) close() {
}

func serve(eventHandler EventHandler, listeners *listener, options *Options) error {
	return ErrUnsupportedPlatform
}
