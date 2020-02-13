// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"net"

	"github.com/panjf2000/gnet/pool/bytebuffer"
	prb "github.com/panjf2000/gnet/pool/ringbuffer"
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
	in *bytebuffer.ByteBuffer
}

type udpIn struct {
	c *stdConn
}

type stdConn struct {
	ctx           interface{}            // user-defined context
	conn          net.Conn               // original connection
	loop          *eventloop             // owner event-loop
	done          int32                  // 0: attached, 1: closed
	buffer        *bytebuffer.ByteBuffer // reuse memory of inbound data as a temporary buffer
	codec         ICodec                 // codec for TCP
	localAddr     net.Addr               // local server addr
	remoteAddr    net.Addr               // remote peer addr
	byteBuffer    *bytebuffer.ByteBuffer // bytes buffer for buffering current packet and data in ring-buffer
	inboundBuffer *ringbuffer.RingBuffer // buffer for data from client
}

func newTCPConn(conn net.Conn, el *eventloop) *stdConn {
	return &stdConn{
		conn:          conn,
		loop:          el,
		codec:         el.codec,
		inboundBuffer: prb.Get(),
	}
}

func (c *stdConn) releaseTCP() {
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	prb.Put(c.inboundBuffer)
	c.inboundBuffer = nil
	bytebuffer.Put(c.buffer)
	c.buffer = nil
}

func newUDPConn(el *eventloop, localAddr, remoteAddr net.Addr, buf *bytebuffer.ByteBuffer) *stdConn {
	return &stdConn{
		loop:       el,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		buffer:     buf,
	}
}

func (c *stdConn) releaseUDP() {
	c.ctx = nil
	c.localAddr = nil
	bytebuffer.Put(c.buffer)
	c.buffer = nil
}

func (c *stdConn) read() ([]byte, error) {
	return c.codec.Decode(c)
}

// ================================= Public APIs of gnet.Conn =================================

func (c *stdConn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		if c.buffer.Len() == 0 {
			return nil
		}
		return c.buffer.Bytes()
	}
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = c.inboundBuffer.WithByteBuffer(c.buffer.Bytes())
	return c.byteBuffer.Bytes()
}

func (c *stdConn) ResetBuffer() {
	c.buffer.Reset()
	c.inboundBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

func (c *stdConn) ShiftN(n int) (size int) {
	tempBufferLen := c.buffer.Len()
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.ResetBuffer()
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		if n == tempBufferLen {
			c.buffer.Reset()
		} else {
			c.buffer.B = c.buffer.B[n:]
		}
		return
	}
	c.byteBuffer.B = c.byteBuffer.B[n:]
	if c.byteBuffer.Len() == 0 {
		bytebuffer.Put(c.byteBuffer)
		c.byteBuffer = nil
	}
	if inBufferLen >= n {
		c.inboundBuffer.Shift(n)
		return
	}
	c.inboundBuffer.Reset()

	restSize := n - inBufferLen
	if restSize == tempBufferLen {
		c.buffer.Reset()
	} else {
		c.buffer.B = c.buffer.B[restSize:]
	}
	return
}

func (c *stdConn) ReadN(n int) (size int, buf []byte) {
	tempBufferLen := c.buffer.Len()
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen+tempBufferLen < n || n <= 0 {
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer.B[:n]
		if n == tempBufferLen {
			c.buffer.Reset()
		} else {
			c.buffer.B = c.buffer.B[n:]
		}
		return
	}
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
	buf = append(buf, c.buffer.B[:restSize]...)
	if restSize == tempBufferLen {
		c.buffer.Reset()
	} else {
		c.buffer.B = c.buffer.B[restSize:]
	}
	return
}

func (c *stdConn) BufferLength() int {
	return c.inboundBuffer.Length() + c.buffer.Len()
}

func (c *stdConn) AsyncWrite(buf []byte) {
	if encodedBuf, err := c.codec.Encode(c, buf); err == nil {
		c.loop.ch <- func() error {
			_, _ = c.conn.Write(encodedBuf)
			return nil
		}
	}
}

func (c *stdConn) SendTo(buf []byte) {
	_, _ = c.loop.svr.ln.pconn.WriteTo(buf, c.remoteAddr)
}

func (c *stdConn) Wake() error {
	c.loop.ch <- wakeReq{c}
	return nil
}

func (c *stdConn) Close() error {
	c.loop.ch <- stderr{c, nil}
	return nil
}

func (c *stdConn) Context() interface{}       { return c.ctx }
func (c *stdConn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *stdConn) LocalAddr() net.Addr        { return c.localAddr }
func (c *stdConn) RemoteAddr() net.Addr       { return c.remoteAddr }
