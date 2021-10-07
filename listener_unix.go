// Copyright (c) 2019 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux || freebsd || dragonfly || darwin
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
