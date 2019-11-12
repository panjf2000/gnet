// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

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
	cache          []byte                 // reuse memory of inbound data
	opened         bool                   // connection opened event fired
	action         Action                 // next user action
	localAddr      net.Addr               // local addr
	remoteAddr     net.Addr               // remote addr
	inboundBuffer  *ringbuffer.RingBuffer // buffer for data from client
	outboundBuffer *ringbuffer.RingBuffer // buffer for data that is ready to write to client
}

func newConn(fd int, lp *loop, sa unix.Sockaddr) *conn {
	return &conn{
		fd:             fd,
		loop:           lp,
		sa:             sa,
		inboundBuffer:  lp.svr.bytesPool.Get().(*ringbuffer.RingBuffer),
		outboundBuffer: lp.svr.bytesPool.Get().(*ringbuffer.RingBuffer),
	}
}

func (c *conn) release() {
	c.opened = false
	c.sa = nil
	c.ctx = nil
	c.cache = nil
	c.localAddr = nil
	c.remoteAddr = nil
	c.inboundBuffer.Reset()
	c.outboundBuffer.Reset()
	c.loop.svr.bytesPool.Put(c.inboundBuffer)
	c.loop.svr.bytesPool.Put(c.outboundBuffer)
	c.inboundBuffer = nil
	c.outboundBuffer = nil
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

func (c *conn) sendTo(buf []byte, sa unix.Sockaddr) {
	_ = unix.Sendto(c.fd, buf, 0, sa)
}

// ================================= Public APIs of gnet.Conn =================================

func (c *conn) ReadFrame() []byte {
	buf, _ := c.loop.svr.codec.Decode(c)
	return buf
}

func (c *conn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		return c.cache
	}
	head, _ := c.inboundBuffer.LazyReadAll()
	return append(head, c.cache...)
}

func (c *conn) ResetBuffer() {
	c.cache = c.cache[:0]
	c.inboundBuffer.Reset()
}

func (c *conn) ReadN(n int) (size int, buf []byte) {
	oneOffBufferLen := len(c.cache)
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen+oneOffBufferLen < n {
		return
	}
	if c.inboundBuffer.IsEmpty() {
		size = n
		buf = c.cache[:n]
		if n == oneOffBufferLen {
			c.cache = c.cache[:0]
		} else {
			c.cache = c.cache[n:]
		}
		return
	}
	size = n
	buf, tail := c.inboundBuffer.LazyRead(n)
	if tail != nil {
		buf = append(buf, tail...)
	}
	if inBufferLen >= n {
		c.inboundBuffer.Shift(n)
		return
	}
	c.inboundBuffer.Reset()

	restSize := n - inBufferLen
	buf = append(buf, c.cache[:restSize]...)
	if restSize == oneOffBufferLen {
		c.cache = c.cache[:0]
	} else {
		c.cache = c.cache[restSize:]
	}
	return
}

func (c *conn) InboundBuffer() *ringbuffer.RingBuffer {
	return c.inboundBuffer
}

func (c *conn) OutboundBuffer() *ringbuffer.RingBuffer {
	return c.outboundBuffer
}

func (c *conn) BufferLength() int {
	return c.inboundBuffer.Length() + len(c.cache)
}

func (c *conn) AsyncWrite(buf []byte) {
	if encodedBuf, err := c.loop.svr.codec.Encode(buf); err == nil {
		_ = c.loop.poller.Trigger(func() error {
			if c.opened {
				c.write(encodedBuf)
			}
			return nil
		})
	}
}

func (c *conn) Wake() {
	if c.loop != nil {
		sniffError(c.loop.poller.Trigger(func() error {
			return c.loop.loopWake(c)
		}))
	}
}

//func (c *conn) ShiftN(n int) {
//	c.inboundBuffer.Shift(n)
//}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
