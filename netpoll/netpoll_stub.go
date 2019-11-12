// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build !linux,!darwin,!dragonfly,!freebsd,!netbsd

package netpoll

import (
	"errors"
	"net"
)

// SetKeepAlive sets the keepalive for the connection.
func SetKeepAlive(fd, secs int) error {
	// OpenBSD has no user-settable per-socket TCP keepalive options.
	return nil
}

// ReusePortListenPacket returns a net.PacketConn for UDP.
func ReusePortListenPacket(proto, addr string) (net.PacketConn, error) {
	return nil, errors.New("reuseport is not available")
}

// ReusePortListen returns a net.Listener for TCP.
func ReusePortListen(proto, addr string) (net.Listener, error) {
	return nil, errors.New("reuseport is not available")
}
