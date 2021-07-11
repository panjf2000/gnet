// Copyright (c) 2020 Andy Pan
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
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
)

func getUnixSockaddr(proto, addr string) (sa unix.Sockaddr, family int, unixAddr *net.UnixAddr, err error) {
	unixAddr, err = net.ResolveUnixAddr(proto, addr)
	if err != nil {
		return
	}

	switch unixAddr.Network() {
	case "unix":
		sa, family = &unix.SockaddrUnix{Name: unixAddr.Name}, unix.AF_UNIX
	default:
		err = errors.ErrUnsupportedUDSProtocol
	}

	return
}

// udsSocket creates an endpoint for communication and returns a file descriptor that refers to that endpoint.
// Argument `reusePort` indicates whether the SO_REUSEPORT flag will be assigned.
func udsSocket(proto, addr string, sockopts ...Option) (fd int, netAddr net.Addr, err error) {
	var (
		family   int
		sockaddr unix.Sockaddr
	)

	if sockaddr, family, netAddr, err = getUnixSockaddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_STREAM, 0); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}
	defer func() {
		if err != nil {
			_ = unix.Close(fd)
		}
	}()

	for _, sockopt := range sockopts {
		if err = sockopt.SetSockopt(fd, sockopt.Opt); err != nil {
			return
		}
	}

	if err = os.NewSyscallError("bind", unix.Bind(fd, sockaddr)); err != nil {
		return
	}

	// Set backlog size to the maximum.
	err = os.NewSyscallError("listen", unix.Listen(fd, listenerBacklogMaxSize))

	return
}
