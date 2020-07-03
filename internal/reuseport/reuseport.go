// copyright (c) 2020 Andy Pan
// Copyright (C) 2017 Max Riveiro
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
// documentation files (the "Software"), to deal in the Software without restriction,
// including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of
// the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
// WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package reuseport provides a function that returns fd and net.Listener powered
// by a net.FileListener with a SO_REUSEPORT option set to the socket.

// +build linux freebsd dragonfly darwin

package reuseport

import (
	"errors"
	"net"
)

var errUnsupportedProtocol = errors.New("only unix, tcp/tcp4/tcp6, udp/udp4/udp6 are supported")

// TCPSocket calls tcpReusablePort.
func TCPSocket(proto, addr string, reusePort bool) (int, net.Addr, error) {
	return tcpReusablePort(proto, addr, reusePort)
}

// UDPSocket calls udpReusablePort.
func UDPSocket(proto, addr string, reusePort bool) (int, net.Addr, error) {
	return udpReusablePort(proto, addr, reusePort)
}

// UnixSocket calls udsReusablePort.
func UnixSocket(proto, addr string, reusePort bool) (int, net.Addr, error) {
	return udsReusablePort(proto, addr, reusePort)
}
