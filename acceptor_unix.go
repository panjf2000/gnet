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

package gnet

import (
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/internal/socket"
	"github.com/panjf2000/gnet/logging"
)

func (svr *server) acceptNewConnection(_ netpoll.IOEvent) error {
	nfd, sa, err := unix.Accept(svr.ln.fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		svr.opts.Logger.Errorf("Accept() fails due to error: %v", err)
		return errors.ErrAcceptSocket
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if svr.opts.TCPKeepAlive > 0 && svr.ln.network == "tcp" {
		err = socket.SetKeepAlive(nfd, int(svr.opts.TCPKeepAlive/time.Second))
		logging.LogErr(err)
	}

	el := svr.lb.next(netAddr)
	c := newTCPConn(nfd, el, sa, netAddr)

	err = el.poller.UrgentTrigger(el.loopRegister, c)
	if err != nil {
		_ = unix.Close(nfd)
		c.releaseTCP()
	}
	return nil
}

func (el *eventloop) loopAccept(_ netpoll.IOEvent) error {
	if el.ln.network == "udp" {
		return el.loopReadUDP(el.ln.fd)
	}

	nfd, sa, err := unix.Accept(el.ln.fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		el.getLogger().Errorf("Accept() fails due to error: %v", err)
		return os.NewSyscallError("accept", err)
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if el.svr.opts.TCPKeepAlive > 0 && el.svr.ln.network == "tcp" {
		err = socket.SetKeepAlive(nfd, int(el.svr.opts.TCPKeepAlive/time.Second))
		logging.LogErr(err)
	}

	c := newTCPConn(nfd, el, sa, netAddr)
	if err = el.poller.AddRead(c.pollAttachment); err == nil {
		el.connections[c.fd] = c
		return el.loopOpen(c)
	}
	return err
}
