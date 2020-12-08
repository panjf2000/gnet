// Copyright (c) 2019 Andy Pan
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

package gnet

import (
	"net"
	"os"
	"sync"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/internal/reuseport"
	"golang.org/x/sys/unix"
)

type listener struct {
	once          sync.Once
	fd            int
	lnaddr        net.Addr
	reusePort     bool
	addr, network string
}

func (ln *listener) Dup() (int, string, error) {
	return netpoll.Dup(ln.fd)
}

func (ln *listener) normalize() (err error) {
	switch ln.network {
	case "tcp", "tcp4", "tcp6":
		ln.fd, ln.lnaddr, err = reuseport.TCPSocket(ln.network, ln.addr, ln.reusePort)
		ln.network = "tcp"
	case "udp", "udp4", "udp6":
		ln.fd, ln.lnaddr, err = reuseport.UDPSocket(ln.network, ln.addr, ln.reusePort)
		ln.network = "udp"
	case "unix":
		_ = os.RemoveAll(ln.addr)
		ln.fd, ln.lnaddr, err = reuseport.UnixSocket(ln.network, ln.addr, ln.reusePort)
	default:
		err = errors.ErrUnsupportedProtocol
	}
	return
}

func (ln *listener) close() {
	ln.once.Do(
		func() {
			if ln.fd > 0 {
				sniffErrorAndLog(os.NewSyscallError("close", unix.Close(ln.fd)))
			}
			if ln.network == "unix" {
				sniffErrorAndLog(os.RemoveAll(ln.addr))
			}
		})
}

func initListener(network, addr string, reusePort bool) (l *listener, err error) {
	l = &listener{network: network, addr: addr, reusePort: reusePort}
	err = l.normalize()
	return
}
