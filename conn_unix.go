// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"io"
	"net"
	"syscall"

	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

type conn struct {
	fd         int // file descriptor
	lnidx      int // listener index in the server lns list
	inBuf      *ringbuffer.RingBuffer
	outBuf     *ringbuffer.RingBuffer
	sa         unix.Sockaddr // remote socket address
	opened     bool          // connection opened event fired
	action     Action        // next user action
	ctx        interface{}   // user-defined context
	addrIndex  int           // index of listening address
	localAddr  net.Addr      // local addre
	remoteAddr net.Addr      // remote addr
	loop       *loop         // connected loop
}

func (c *conn) sendOut(buf []byte) {
	//if !c.outBuf.IsFull() && !c.outBuf.IsEmpty() {
	//	_, _ = c.outBuf.Write(buf)
	//	return
	//}
	n, err := syscall.Write(c.fd, buf)
	if err != nil {
		_, _ = c.outBuf.Write(buf)
		return
	}

	if n < len(buf) {
		_, _ = c.outBuf.Write(buf[n:])
	}
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) AddrIndex() int             { return c.addrIndex }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *conn) Wake() {
	if c.loop != nil {
		sniffError(c.loop.poller.Trigger(c))
	}
}

type detachedConn struct {
	fd int
}

func (c *detachedConn) Close() error {
	err := unix.Close(c.fd)
	if err != nil {
		return err
	}
	c.fd = -1
	return nil
}

func (c *detachedConn) Read(p []byte) (n int, err error) {
	n, err = unix.Read(c.fd, p)
	if err != nil {
		return n, err
	}
	if n == 0 {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	return n, nil
}

func (c *detachedConn) Write(p []byte) (n int, err error) {
	n = len(p)
	for len(p) > 0 {
		nn, err := unix.Write(c.fd, p)
		if err != nil {
			return n, err
		}
		p = p[nn:]
	}
	return n, nil
}
