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
	inBuf      *ringbuffer.RingBuffer
	outBuf     *ringbuffer.RingBuffer
	sa         unix.Sockaddr // remote socket address
	opened     bool          // connection opened event fired
	action     Action        // next user action
	ctx        interface{}   // user-defined context
	localAddr  net.Addr      // local addre
	remoteAddr net.Addr      // remote addr
	loop       *loop         // connected loop
}

func (c *conn) ReadPair() ([]byte, []byte) {
	return c.inBuf.PreReadAll()
}

func (c *conn) ReadBytes() []byte {
	return c.inBuf.Bytes()
}

func (c *conn) AdvanceBuffer(n int) {
	c.inBuf.Advance(n)
}

func (c *conn) ResetBuffer() {
	c.inBuf.Reset()
}

func (c *conn) AsyncWrite(buf []byte) {
	_ = c.loop.poller.Trigger(func() {
		c.write(buf)
		ringbuffer.Recycle(buf)
	})
}

func (c *conn) open(buf []byte) {
	n, err := syscall.Write(c.fd, buf)
	if err != nil {
		_, _ = c.outBuf.Write(buf)
		return
	}

	if n < len(buf) {
		_, _ = c.outBuf.Write(buf[n:])
	}
}

func (c *conn) write(buf []byte) {
	n, err := syscall.Write(c.fd, buf)
	if err != nil {
		_, _ = c.outBuf.Write(buf)
		c.loop.poller.ModReadWrite(c.fd)
		return
	}

	if n < len(buf) {
		_, _ = c.outBuf.Write(buf[n:])
		c.loop.poller.ModReadWrite(c.fd)
	}
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *conn) Wake() {
	if c.loop != nil {
		sniffError(c.loop.poller.Trigger(c))
	}
}
