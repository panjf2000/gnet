// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"github.com/panjf2000/gnet/pool"
	"net"

	"github.com/panjf2000/gnet/ringbuffer"
)

type stderr struct {
	c   *stdConn
	err error
}

type wakeReq struct {
	c *stdConn
}

type tcpIn struct {
	c  *stdConn
	in []byte
}

type udpIn struct {
	c *stdConn
}

type stdConn struct {
	ctx           interface{}            // user-defined context
	conn          net.Conn               // original connection
	loop          *loop                  // owner loop
	done          int32                  // 0: attached, 1: closed
	cache         []byte                 // reuse memory of inbound data
	localAddr     net.Addr               // local server addr
	remoteAddr    net.Addr               // remote peer addr
	inboundBuffer *ringbuffer.RingBuffer // buffer for data from client
}

func newConn(conn net.Conn, lp *loop) *stdConn {
	return &stdConn{
		conn:          conn,
		loop:          lp,
		inboundBuffer: lp.svr.bytesPool.Get().(*ringbuffer.RingBuffer),
	}
}

func (c *stdConn) release() {
	//c.conn = nil
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	c.inboundBuffer.Reset()
	c.loop.svr.bytesPool.Put(c.inboundBuffer)
	c.inboundBuffer = nil
	pool.PutBytes(c.cache)
	c.cache = nil
}

// ================================= Public APIs of gnet.Conn =================================

func (c *stdConn) ReadFrame() []byte {
	buf, _ := c.loop.svr.codec.Decode(c)
	return buf
}

func (c *stdConn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		return c.cache
	}
	head, _ := c.inboundBuffer.LazyReadAll()
	return append(head, c.cache...)
}

func (c *stdConn) ResetBuffer() {
	c.cache = c.cache[:0]
	c.inboundBuffer.Reset()
}

func (c *stdConn) ReadN(n int) (size int, buf []byte) {
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

func (c *stdConn) InboundBuffer() *ringbuffer.RingBuffer {
	return c.inboundBuffer
}

func (c *stdConn) OutboundBuffer() *ringbuffer.RingBuffer {
	return nil
}

func (c *stdConn) BufferLength() int {
	return c.inboundBuffer.Length() + len(c.cache)
}

func (c *stdConn) AsyncWrite(buf []byte) {
	if encodedBuf, err := c.loop.svr.codec.Encode(buf); err == nil {
		c.loop.ch <- func() error {
			_, _ = c.conn.Write(encodedBuf)
			return nil
		}
	}
}

func (c *stdConn) Context() interface{}       { return c.ctx }
func (c *stdConn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *stdConn) LocalAddr() net.Addr        { return c.localAddr }
func (c *stdConn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *stdConn) Wake()                      { c.loop.ch <- wakeReq{c} }
