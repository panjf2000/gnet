// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly windows

package netpoll

import (
	"net"

	"github.com/libp2p/go-reuseport"
)

// ReusePortListenPacket returns a net.PacketConn for UDP.
func ReusePortListenPacket(proto, addr string) (net.PacketConn, error) {
	return reuseport.ListenPacket(proto, addr)
}

// ReusePortListen returns a net.Listener for TCP.
func ReusePortListen(proto, addr string) (net.Listener, error) {
	return reuseport.Listen(proto, addr)
}
