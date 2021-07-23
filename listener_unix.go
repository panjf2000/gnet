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
	"strings"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/internal/socket"
	"github.com/panjf2000/gnet/logging"
)

type listener struct {
	once           sync.Once
	fd             int
	lnaddr         net.Addr
	addr, network  string
	sockopts       []socket.Option
	pollAttachment *netpoll.PollAttachment // listener attachment for poller
}

func (ln *listener) packPollAttachment(handler netpoll.PollEventHandler) *netpoll.PollAttachment {
	ln.pollAttachment = &netpoll.PollAttachment{FD: ln.fd, Callback: handler}
	return ln.pollAttachment
}

func (ln *listener) dup() (int, string, error) {
	return netpoll.Dup(ln.fd)
}

func (ln *listener) normalize() (err error) {
	switch ln.network {
	case "tcp", "tcp4", "tcp6":
		ln.fd, ln.lnaddr, err = socket.TCPSocket(ln.network, ln.addr, ln.sockopts...)
		ln.network = "tcp"
	case "udp", "udp4", "udp6":
		ln.fd, ln.lnaddr, err = socket.UDPSocket(ln.network, ln.addr, ln.sockopts...)
		ln.network = "udp"
	case "unix":
		_ = os.RemoveAll(ln.addr)
		ln.fd, ln.lnaddr, err = socket.UnixSocket(ln.network, ln.addr, ln.sockopts...)
	default:
		err = errors.ErrUnsupportedProtocol
	}
	return
}

func (ln *listener) close() {
	ln.once.Do(
		func() {
			if ln.fd > 0 {
				logging.LogErr(os.NewSyscallError("close", unix.Close(ln.fd)))
			}
			if ln.network == "unix" {
				logging.LogErr(os.RemoveAll(ln.addr))
			}
		})
}

func initListener(network, addr string, options *Options) (l *listener, err error) {
	var sockopts []socket.Option
	if options.ReusePort || strings.HasPrefix(network, "udp") {
		sockopt := socket.Option{SetSockopt: socket.SetReuseport, Opt: 1}
		sockopts = append(sockopts, sockopt)
	}
	if options.TCPNoDelay == TCPNoDelay && strings.HasPrefix(network, "tcp") {
		sockopt := socket.Option{SetSockopt: socket.SetNoDelay, Opt: 1}
		sockopts = append(sockopts, sockopt)
	}
	if options.SocketRecvBuffer > 0 {
		sockopt := socket.Option{SetSockopt: socket.SetRecvBuffer, Opt: options.SocketRecvBuffer}
		sockopts = append(sockopts, sockopt)
	}
	if options.SocketSendBuffer > 0 {
		sockopt := socket.Option{SetSockopt: socket.SetSendBuffer, Opt: options.SocketSendBuffer}
		sockopts = append(sockopts, sockopt)
	}
	l = &listener{network: network, addr: addr, sockopts: sockopts}
	err = l.normalize()
	return
}
