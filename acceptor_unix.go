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
	"os"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/socket"
	"golang.org/x/sys/unix"
)

func (svr *server) acceptNewConnection(fd int) error {
	nfd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return errors.ErrAcceptSocket
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	el := svr.lb.next(netAddr)
	c := newTCPConn(nfd, el, sa, netAddr)

	err = el.poller.Trigger(func() (err error) {
		if err = el.poller.AddRead(nfd); err != nil {
			_ = unix.Close(nfd)
			c.releaseTCP()
			return
		}
		el.connections[nfd] = c
		err = el.loopOpen(c)
		return
	})
	if err != nil {
		_ = unix.Close(nfd)
		c.releaseTCP()
	}
	return nil
}
