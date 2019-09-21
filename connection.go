// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
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

func (c *conn) Read() (top, tail []byte) {
	top, tail = c.inBuf.PreReadAll()
	return
}

func (c *conn) Advance(n int) {
	c.inBuf.Advance(n)
}

func (c *conn) ResetBuffer() {
	c.inBuf.Reset()
}

func (c *conn) Write(buf []byte) {
	_, _ = c.outBuf.Write(buf)
}

func (c *conn) write() {
	//buf := c.outBuf.Bytes()
	top, _ := c.outBuf.PreReadAll()
	n, err := syscall.Write(c.fd, top)
	if err != nil {
		return
	}
	c.outBuf.Advance(n)
	//ringbuffer.Recycle(buf)
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
