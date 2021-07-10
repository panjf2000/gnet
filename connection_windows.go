// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
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

package gnet

import (
	"net"
	"sync"

	"github.com/panjf2000/gnet/pool/bytebuffer"
	prb "github.com/panjf2000/gnet/pool/ringbuffer"
	"github.com/panjf2000/gnet/ringbuffer"
)

type stderr struct {
	c   *stdConn
	err error
}

type signalTask struct {
	run func(*stdConn) error
	c   *stdConn
}

type dataTask struct {
	run func([]byte) (int, error)
	buf []byte
}

type tcpConn struct {
	c  *stdConn
	bb *bytebuffer.ByteBuffer
}

type udpConn struct {
	c *stdConn
}

var (
	signalTaskPool = sync.Pool{New: func() interface{} { return new(signalTask) }}
	dataTaskPool   = sync.Pool{New: func() interface{} { return new(dataTask) }}
)

type stdConn struct {
	ctx           interface{}            // user-defined context
	conn          net.Conn               // original connection
	loop          *eventloop             // owner event-loop
	buffer        *bytebuffer.ByteBuffer // reuse memory of inbound data as a temporary buffer
	codec         ICodec                 // codec for TCP
	localAddr     net.Addr               // local server addr
	remoteAddr    net.Addr               // remote peer addr
	byteBuffer    *bytebuffer.ByteBuffer // bytes buffer for buffering current packet and data in ring-buffer
	inboundBuffer *ringbuffer.RingBuffer // buffer for data from client
}

func packTCPConn(c *stdConn, buf []byte) *tcpConn {
	packet := &tcpConn{c: c}
	packet.bb = bytebuffer.Get()
	_, _ = packet.bb.Write(buf)
	return packet
}

func packUDPConn(c *stdConn, buf []byte) *udpConn {
	_, _ = c.buffer.Write(buf)
	packet := &udpConn{c: c}
	return packet
}

func newTCPConn(conn net.Conn, el *eventloop) (c *stdConn) {
	c = &stdConn{
		conn:          conn,
		loop:          el,
		codec:         el.svr.codec,
		inboundBuffer: prb.Get(),
	}
	c.localAddr = el.svr.ln.lnaddr
	c.remoteAddr = c.conn.RemoteAddr()

	var (
		ok bool
		tc *net.TCPConn
	)
	if tc, ok = conn.(*net.TCPConn); !ok {
		return
	}
	var noDelay bool
	switch el.svr.opts.TCPNoDelay {
	case TCPNoDelay:
		noDelay = true
	case TCPDelay:
	}
	_ = tc.SetNoDelay(noDelay)
	if el.svr.opts.TCPKeepAlive > 0 {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(el.svr.opts.TCPKeepAlive)
	}
	return
}

func (c *stdConn) releaseTCP() {
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	c.conn = nil
	prb.Put(c.inboundBuffer)
	c.inboundBuffer = ringbuffer.EmptyRingBuffer
	bytebuffer.Put(c.buffer)
	c.buffer = nil
}

func newUDPConn(el *eventloop, localAddr, remoteAddr net.Addr) *stdConn {
	return &stdConn{
		loop:       el,
		buffer:     bytebuffer.Get(),
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
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

func (c *stdConn) write(data []byte) (n int, err error) {
	if c.conn != nil {
		n, err = c.conn.Write(data)
	}
	return
}

// ================================= Public APIs of gnet.Conn =================================

func (c *stdConn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		return c.buffer.Bytes()
	}
	c.byteBuffer = c.inboundBuffer.WithByteBuffer(c.buffer.Bytes())
	return c.byteBuffer.Bytes()
}

func (c *stdConn) ResetBuffer() {
	c.buffer.Reset()
	c.inboundBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

func (c *stdConn) ReadN(n int) (size int, buf []byte) {
	inBufferLen := c.inboundBuffer.Length()
	tempBufferLen := c.buffer.Len()
	if totalLen := inBufferLen + tempBufferLen; totalLen < n || n <= 0 {
		n = totalLen
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer.B[:n]
		return
	}
	head, tail := c.inboundBuffer.Peek(n)
	c.byteBuffer = bytebuffer.Get()
	_, _ = c.byteBuffer.Write(head)
	_, _ = c.byteBuffer.Write(tail)
	if inBufferLen >= n {
		buf = c.byteBuffer.Bytes()
		return
	}

	restSize := n - inBufferLen
	_, _ = c.byteBuffer.Write(c.buffer.B[:restSize])
	buf = c.byteBuffer.Bytes()
	return
}

func (c *stdConn) ShiftN(n int) (size int) {
	inBufferLen := c.inboundBuffer.Length()
	tempBufferLen := c.buffer.Len()
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.ResetBuffer()
		size = inBufferLen + tempBufferLen
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		c.buffer.B = c.buffer.B[n:]
		return
	}

	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil

	if inBufferLen >= n {
		c.inboundBuffer.Discard(n)
		return
	}
	c.inboundBuffer.Reset()

	restSize := n - inBufferLen
	c.buffer.B = c.buffer.B[restSize:]
	return
}

func (c *stdConn) BufferLength() int {
	return c.inboundBuffer.Length() + c.buffer.Len()
}

func (c *stdConn) AsyncWrite(buf []byte) (err error) {
	var encodedBuf []byte
	if encodedBuf, err = c.codec.Encode(c, buf); err == nil {
		task := dataTaskPool.Get().(*dataTask)
		task.run = c.write
		task.buf = encodedBuf
		c.loop.ch <- task
	}
	return
}

func (c *stdConn) SendTo(buf []byte) (err error) {
	_, err = c.loop.svr.ln.pconn.WriteTo(buf, c.remoteAddr)
	return
}

func (c *stdConn) Wake() error {
	task := signalTaskPool.Get().(*signalTask)
	task.run = c.loop.loopWake
	task.c = c
	c.loop.ch <- task
	return nil
}

func (c *stdConn) Close() error {
	task := signalTaskPool.Get().(*signalTask)
	task.run = c.loop.loopCloseConn
	task.c = c
	c.loop.ch <- task
	return nil
}

func (c *stdConn) Context() interface{}       { return c.ctx }
func (c *stdConn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *stdConn) LocalAddr() net.Addr        { return c.localAddr }
func (c *stdConn) RemoteAddr() net.Addr       { return c.remoteAddr }
