// Copyright (c) 2021 Andy Pan
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
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/socket"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

func (eng *engine) accept(fd int, _ netpoll.IOEvent) error {
	nfd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		eng.opts.Logger.Errorf("Accept() failed due to error: %v", err)
		return errors.ErrAcceptSocket
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if eng.opts.TCPKeepAlive > 0 && eng.ln.network == "tcp" {
		err = socket.SetKeepAlivePeriod(nfd, int(eng.opts.TCPKeepAlive.Seconds()))
		logging.Error(err)
	}

	el := eng.lb.next(remoteAddr)
	c := newTCPConn(nfd, el, sa, el.ln.addr, remoteAddr)

	err = el.poller.UrgentTrigger(el.register, c)
	if err != nil {
		eng.opts.Logger.Errorf("UrgentTrigger() failed due to error: %v", err)
		_ = unix.Close(nfd)
		c.releaseTCP()
	}
	return nil
}

func (el *eventloop) accept(fd int, ev netpoll.IOEvent) error {
	if el.ln.network == "udp" {
		return el.readUDP(fd, ev)
	}

	nfd, sa, err := unix.Accept(el.ln.fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		el.getLogger().Errorf("Accept() failed due to error: %v", err)
		return os.NewSyscallError("accept", err)
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if el.engine.opts.TCPKeepAlive > 0 && el.ln.network == "tcp" {
		err = socket.SetKeepAlivePeriod(nfd, int(el.engine.opts.TCPKeepAlive/time.Second))
		logging.Error(err)
	}

	c := newTCPConn(nfd, el, sa, el.ln.addr, remoteAddr)
	if err = el.poller.AddRead(c.pollAttachment); err != nil {
		return err
	}
	el.connections[c.fd] = c
	return el.open(c)
}
