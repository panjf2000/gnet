// Copyright (c) 2021 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// +build linux freebsd dragonfly darwin

package socket

import (
	"os"

	"golang.org/x/sys/unix"
)

// SetNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
//
// The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
func SetNoDelay(fd, noDelay int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, noDelay))
}

// SetRecvBuffer sets the size of the operating system's
// receive buffer associated with the connection.
func SetRecvBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size)
}

// SetSendBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
func SetSendBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size)
}

// SetReuseport enables SO_REUSEPORT option on socket.
func SetReuseport(fd, reusePort int) error {
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, reusePort)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, reusePort))
}

// SetIPv6Only restricts a IPv6 socket to only process IPv6 requests or both IPv4 and IPv6 requests.
func SetIPv6Only(fd, ipv6only int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, ipv6only)
}
