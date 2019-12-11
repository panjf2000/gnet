// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"net"

	"github.com/panjf2000/gnet/pool/bytes"
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
	codec         ICodec                 // codec for TCP
	localAddr     net.Addr               // local server addr
	remoteAddr    net.Addr               // remote peer addr
	inboundBuffer *ringbuffer.RingBuffer // buffer for data from client
}

func newTCPConn(conn net.Conn, lp *loop) *stdConn {
	return &stdConn{
		conn:          conn,
		loop:          lp,
		codec:         lp.codec,
		inboundBuffer: prb.Get(),
	}
}

func (c *stdConn) releaseTCP() {
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	prb.Put(c.inboundBuffer)
	c.inboundBuffer = nil
	bytes.Put(c.cache)
	c.cache = nil
}

func newUDPConn(lp *loop, localAddr, remoteAddr net.Addr, buf []byte) *stdConn {
	return &stdConn{
		loop:       lp,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		cache:      buf,
	}
}

func (c *stdConn) releaseUDP() {
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	bytes.Put(c.cache)
	c.cache = nil
}

// ================================= Public APIs of gnet.Conn =================================

func (c *stdConn) ReadFromUDP() []byte {
	return c.cache
}

func (c *stdConn) ReadFrame() []byte {
	buf, _ := c.codec.Decode(c)
	return buf
}

func (c *stdConn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		return c.cache
	}
	return c.inboundBuffer.WithBytes(c.cache)
}

func (c *stdConn) ResetBuffer() {
	c.cache = c.cache[:0]
	c.inboundBuffer.Reset()
}

func (c *stdConn) ShiftN(n int) (size int) {
	oneOffBufferLen := len(c.cache)
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen+oneOffBufferLen < n || n <= 0 {
		c.ResetBuffer()
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		if n == oneOffBufferLen {
			c.cache = c.cache[:0]
		} else {
			c.cache = c.cache[n:]
		}
		return
	}
	if inBufferLen >= n {
		c.inboundBuffer.Shift(n)
		return
	}
	c.inboundBuffer.Reset()

	restSize := n - inBufferLen
	if restSize == oneOffBufferLen {
		c.cache = c.cache[:0]
	} else {
		c.cache = c.cache[restSize:]
	}
	return
}

func (c *stdConn) ReadN(n int) (size int, buf []byte) {
	oneOffBufferLen := len(c.cache)
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen+oneOffBufferLen < n || n <= 0 {
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		buf = c.cache[:n]
		if n == oneOffBufferLen {
			c.cache = c.cache[:0]
		} else {
			c.cache = c.cache[n:]
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
	buf = append(buf, c.cache[:restSize]...)
	if restSize == oneOffBufferLen {
		c.cache = c.cache[:0]
	} else {
		c.cache = c.cache[restSize:]
	}
	return
}

//func (c *stdConn) InboundBuffer() *ringbuffer.RingBuffer {
//	return c.inboundBuffer
//}

func (c *stdConn) BufferLength() int {
	return c.inboundBuffer.Length() + len(c.cache)
}

func (c *stdConn) AsyncWrite(buf []byte) {
	if encodedBuf, err := c.codec.Encode(c, buf); err == nil {
		c.loop.ch <- func() error {
			_, _ = c.conn.Write(encodedBuf)
			bytes.Put(buf)
			return nil
		}
	}
}

func (c *stdConn) SendTo(buf []byte) {
	_, _ = c.loop.svr.ln.pconn.WriteTo(buf, c.remoteAddr)
}

func (c *stdConn) Context() interface{}       { return c.ctx }
func (c *stdConn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *stdConn) LocalAddr() net.Addr        { return c.localAddr }
func (c *stdConn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *stdConn) Wake()                      { c.loop.ch <- wakeReq{c} }
