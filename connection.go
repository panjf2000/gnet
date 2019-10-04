// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"net"

	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

type conn struct {
	fd             int                    // file descriptor
	sa             unix.Sockaddr          // remote socket address
	ctx            interface{}            // user-defined context
	loop           *loop                  // connected loop
	extra          []byte                 // reuse memory of inbound data
	opened         bool                   // connection opened event fired
	action         Action                 // next user action
	localAddr      net.Addr               // local addre
	remoteAddr     net.Addr               // remote addr
	inboundBuffer  *ringbuffer.RingBuffer // buffer for data from client
	outboundBuffer *ringbuffer.RingBuffer // buffer for data that is ready to write to client
}

func (c *conn) ReadPair() (top, tail []byte) {
	if c.inboundBuffer.IsEmpty() {
		top = c.extra
		return
	}
	top, _ = c.inboundBuffer.PreReadAll()
	tail = c.extra
	return
}

func (c *conn) ReadBytes() []byte {
	return c.inboundBuffer.WithBytes(c.extra)
}

func (c *conn) ResetBuffer() {
	c.inboundBuffer.Reset()
}

func (c *conn) AsyncWrite(buf []byte) {
	_ = c.loop.poller.Trigger(func() error {
		c.write(buf)
		ringbuffer.Recycle(buf)
		return nil
	})
}

func (c *conn) open(buf []byte) {
	n, err := unix.Write(c.fd, buf)
	if err != nil {
		_, _ = c.outboundBuffer.Write(buf)
		return
	}

	if n < len(buf) {
		_, _ = c.outboundBuffer.Write(buf[n:])
	}
}

func (c *conn) write(buf []byte) {
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(buf)
		return
	}
	n, err := unix.Write(c.fd, buf)
	if err != nil {
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Write(buf)
			_ = c.loop.poller.ModReadWrite(c.fd)
			return
		}
		_ = c.loop.loopCloseConn(c, err)
		return
	}
	if n < len(buf) {
		_, _ = c.outboundBuffer.Write(buf[n:])
		_ = c.loop.poller.ModReadWrite(c.fd)
	}
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *conn) SendTo(buf []byte, sa unix.Sockaddr) {
	_ = unix.Sendto(c.fd, buf, 0, sa)
}

//func (c *conn) Wake() {
//	if c.loop != nil {
//		sniffError(c.loop.poller.Trigger(c))
//	}
//}
