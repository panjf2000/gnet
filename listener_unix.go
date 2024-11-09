// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package gnet

import (
	"net"
	"os"
	"strings"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/socket"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type listener struct {
	once             sync.Once
	fd               int
	addr             net.Addr
	address, network string
	sockOptInts      []socket.Option[int]
	sockOptStrs      []socket.Option[string]
	pollAttachment   *netpoll.PollAttachment // listener attachment for poller
}

func (ln *listener) packPollAttachment(handler netpoll.PollEventHandler) *netpoll.PollAttachment {
	ln.pollAttachment = &netpoll.PollAttachment{FD: ln.fd, Callback: handler}
	return ln.pollAttachment
}

func (ln *listener) dup() (int, error) {
	return socket.Dup(ln.fd)
}

func (ln *listener) normalize() (err error) {
	switch ln.network {
	case "tcp", "tcp4", "tcp6":
		ln.fd, ln.addr, err = socket.TCPSocket(ln.network, ln.address, true, ln.sockOptInts, ln.sockOptStrs)
		ln.network = "tcp"
	case "udp", "udp4", "udp6":
		ln.fd, ln.addr, err = socket.UDPSocket(ln.network, ln.address, false, ln.sockOptInts, ln.sockOptStrs)
		ln.network = "udp"
	case "unix":
		_ = os.RemoveAll(ln.address)
		ln.fd, ln.addr, err = socket.UnixSocket(ln.network, ln.address, true, ln.sockOptInts, ln.sockOptStrs)
	default:
		err = errors.ErrUnsupportedProtocol
	}
	return
}

func (ln *listener) close() {
	ln.once.Do(
		func() {
			if ln.fd > 0 {
				logging.Error(os.NewSyscallError("close", unix.Close(ln.fd)))
			}
			if ln.network == "unix" {
				logging.Error(os.RemoveAll(ln.address))
			}
		})
}

func initListener(network, addr string, options *Options) (l *listener, err error) {
	var (
		sockOptInts []socket.Option[int]
		sockOptStrs []socket.Option[string]
	)
	if options.ReusePort || strings.HasPrefix(network, "udp") {
		sockOpt := socket.Option[int]{SetSockOpt: socket.SetReuseport, Opt: 1}
		sockOptInts = append(sockOptInts, sockOpt)
	}
	if options.ReuseAddr {
		sockOpt := socket.Option[int]{SetSockOpt: socket.SetReuseAddr, Opt: 1}
		sockOptInts = append(sockOptInts, sockOpt)
	}
	if options.TCPNoDelay == TCPNoDelay && strings.HasPrefix(network, "tcp") {
		sockOpt := socket.Option[int]{SetSockOpt: socket.SetNoDelay, Opt: 1}
		sockOptInts = append(sockOptInts, sockOpt)
	}
	if options.SocketRecvBuffer > 0 {
		sockOpt := socket.Option[int]{SetSockOpt: socket.SetRecvBuffer, Opt: options.SocketRecvBuffer}
		sockOptInts = append(sockOptInts, sockOpt)
	}
	if options.SocketSendBuffer > 0 {
		sockOpt := socket.Option[int]{SetSockOpt: socket.SetSendBuffer, Opt: options.SocketSendBuffer}
		sockOptInts = append(sockOptInts, sockOpt)
	}
	if strings.HasPrefix(network, "udp") {
		udpAddr, err := net.ResolveUDPAddr(network, addr)
		if err == nil && udpAddr.IP.IsMulticast() {
			if sockoptFn := socket.SetMulticastMembership(network, udpAddr); sockoptFn != nil {
				sockOpt := socket.Option[int]{SetSockOpt: sockoptFn, Opt: options.MulticastInterfaceIndex}
				sockOptInts = append(sockOptInts, sockOpt)
			}
		}
	}
	if options.BindToDevice != "" {
		sockOpt := socket.Option[string]{SetSockOpt: socket.SetBindToDevice, Opt: options.BindToDevice}
		sockOptStrs = append(sockOptStrs, sockOpt)
	}
	l = &listener{network: network, address: addr, sockOptInts: sockOptInts, sockOptStrs: sockOptStrs}
	err = l.normalize()
	return
}
