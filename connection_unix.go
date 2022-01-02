// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"

	gio "github.com/panjf2000/gnet/v2/internal/io"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/socket"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/mixedbuffer"
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	rbPool "github.com/panjf2000/gnet/v2/pkg/pool/ringbuffer"
	"github.com/panjf2000/gnet/v2/pkg/ringbuffer"
)

type conn struct {
	fd             int                     // file descriptor
	ctx            interface{}             // user-defined context
	peer           unix.Sockaddr           // remote socket address
	loop           *eventloop              // connected event-loop
	cache          *bbPool.ByteBuffer      // temporary buffer in each event-loop
	buffer         []byte                  // buffer for the latest bytes
	opened         bool                    // connection opened event fired
	packets        [][]byte                // reuse it for multiple byte slices
	isDatagram     bool                    // UDP protocol
	localAddr      net.Addr                // local addr
	remoteAddr     net.Addr                // remote addr
	inboundBuffer  *ringbuffer.RingBuffer  // buffer for leftover data from the peer
	outboundBuffer *mixedbuffer.Buffer     // buffer for data that is eligible to be sent to the peer
	pollAttachment *netpoll.PollAttachment // connection attachment for poller
}

func newTCPConn(fd int, el *eventloop, sa unix.Sockaddr, localAddr, remoteAddr net.Addr) (c *conn) {
	c = &conn{
		fd:             fd,
		peer:           sa,
		loop:           el,
		localAddr:      localAddr,
		remoteAddr:     remoteAddr,
		inboundBuffer:  rbPool.Get(),
		outboundBuffer: mixedbuffer.New(MaxStreamBufferCap),
	}
	c.pollAttachment = netpoll.GetPollAttachment()
	c.pollAttachment.FD, c.pollAttachment.Callback = fd, c.handleEvents
	return
}

func (c *conn) releaseTCP() {
	c.opened = false
	c.peer = nil
	c.ctx = nil
	c.packets = nil
	c.buffer = nil
	if addr, ok := c.localAddr.(*net.TCPAddr); ok && c.localAddr != c.loop.ln.addr {
		bsPool.Put(addr.IP)
	}
	if addr, ok := c.remoteAddr.(*net.TCPAddr); ok {
		bsPool.Put(addr.IP)
	}
	c.localAddr = nil
	c.remoteAddr = nil
	rbPool.Put(c.inboundBuffer)
	c.inboundBuffer = ringbuffer.New(0)
	c.outboundBuffer.Release()
	bbPool.Put(c.cache)
	c.cache = nil
	netpoll.PutPollAttachment(c.pollAttachment)
	c.pollAttachment = nil
}

func newUDPConn(fd int, el *eventloop, localAddr net.Addr, sa unix.Sockaddr, connected bool) (c *conn) {
	c = &conn{
		fd:            fd,
		peer:          sa,
		loop:          el,
		localAddr:     localAddr,
		remoteAddr:    socket.SockaddrToUDPAddr(sa),
		isDatagram:    true,
		inboundBuffer: ringbuffer.New(0),
	}
	if connected {
		c.peer = nil
	}
	return
}

func (c *conn) releaseUDP() {
	c.ctx = nil
	if addr, ok := c.localAddr.(*net.UDPAddr); ok && c.localAddr != c.loop.ln.addr {
		bsPool.Put(addr.IP)
	}
	if addr, ok := c.remoteAddr.(*net.UDPAddr); ok {
		bsPool.Put(addr.IP)
	}
	c.localAddr = nil
	c.remoteAddr = nil
	c.buffer = nil
	netpoll.PutPollAttachment(c.pollAttachment)
	c.pollAttachment = nil
}

func (c *conn) open(buf []byte) error {
	n, err := unix.Write(c.fd, buf)
	if err != nil && err == unix.EAGAIN {
		_, _ = c.outboundBuffer.Write(buf)
		return nil
	}

	if err == nil && n < len(buf) {
		_, _ = c.outboundBuffer.Write(buf[n:])
	}

	return err
}

func (c *conn) write(data []byte) (err error) {
	// If there is pending data in outbound buffer, the current data ought to be appended to the outbound buffer
	// for maintaining the sequence of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(data)
		return
	}

	var n int
	if n, err = unix.Write(c.fd, data); err != nil {
		// A temporary error occurs, append the data to outbound buffer, writing it back to the peer in the next round.
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Write(data)
			err = c.loop.poller.ModReadWrite(c.pollAttachment)
			return
		}
		return c.loop.closeConn(c, os.NewSyscallError("write", err))
	}
	// Failed to send all data back to the peer, buffer the leftover data for the next round.
	if n < len(data) {
		_, _ = c.outboundBuffer.Write(data[n:])
		err = c.loop.poller.ModReadWrite(c.pollAttachment)
	}
	return
}

func (c *conn) writev(bs [][]byte) (err error) {
	defer func() {
		c.packets = c.packets[:0]
	}()

	var sum int
	for _, b := range bs {
		c.packets = append(c.packets, b)
		sum += len(b)
	}

	// If there is pending data in outbound buffer, the current data ought to be appended to the outbound buffer
	// for maintaining the sequence of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Writev(c.packets)
		return
	}

	var n int
	if n, err = gio.Writev(c.fd, c.packets); err != nil {
		// A temporary error occurs, append the data to outbound buffer, writing it back to the peer in the next round.
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Writev(c.packets)
			err = c.loop.poller.ModReadWrite(c.pollAttachment)
			return
		}
		return c.loop.closeConn(c, os.NewSyscallError("write", err))
	}
	// Failed to send all data back to the peer, buffer the leftover data for the next round.
	if n < sum {
		var pos int
		for i := range c.packets {
			np := len(c.packets[i])
			if n < np {
				c.packets[i] = c.packets[i][n:]
				pos = i
				break
			}
			n -= np
		}
		_, _ = c.outboundBuffer.Writev(c.packets[pos:])
		err = c.loop.poller.ModReadWrite(c.pollAttachment)
	}
	return
}

func (c *conn) asyncWrite(itf interface{}) error {
	if !c.opened {
		return nil
	}

	return c.write(itf.([]byte))
}

func (c *conn) asyncWritev(itf interface{}) error {
	if !c.opened {
		return nil
	}

	return c.writev(itf.([][]byte))
}

func (c *conn) sendTo(buf []byte) error {
	if c.peer == nil {
		return unix.Send(c.fd, buf, 0)
	}
	return unix.Sendto(c.fd, buf, 0, c.peer)
}

func (c *conn) resetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
	bbPool.Put(c.cache)
	c.cache = nil
}

// ================================== Non-concurrency-safe API's ==================================

func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		return n, nil
	}
	n, _ = c.inboundBuffer.Read(p)
	n += copy(p[n:], c.buffer)
	return c.Discard(n)
}

func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); totalLen < n || n <= 0 {
		err = gerrors.ErrBufferFull
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer[:n]
		c.buffer = c.buffer[n:]
		return
	}
	head, tail := c.inboundBuffer.Peek(n)
	defer c.inboundBuffer.Discard(n)
	if len(head) >= n {
		return head[:n], err
	}
	c.cache = bbPool.Get()
	_, _ = c.cache.Write(head)
	_, _ = c.cache.Write(tail)
	if inBufferLen >= n {
		return c.cache.B, err
	}

	remaining := n - inBufferLen
	_, _ = c.cache.Write(c.buffer[:remaining])
	c.buffer = c.buffer[remaining:]
	return c.cache.B, err
}

func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); totalLen < n || n <= 0 {
		err = gerrors.ErrBufferFull
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		return c.buffer[:n], err
	}
	head, tail := c.inboundBuffer.Peek(n)
	if len(head) >= n {
		return head[:n], err
	}
	c.cache = bbPool.Get()
	_, _ = c.cache.Write(head)
	_, _ = c.cache.Write(tail)
	if inBufferLen >= n {
		return c.cache.B, err
	}

	remaining := n - inBufferLen
	_, _ = c.cache.Write(c.buffer[:remaining])
	return c.cache.B, err
}

func (c *conn) Discard(n int) (int, error) {
	inBufferLen := c.inboundBuffer.Buffered()
	tempBufferLen := len(c.buffer)
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.resetBuffer()
		return inBufferLen + tempBufferLen, nil
	}
	if c.inboundBuffer.IsEmpty() {
		c.buffer = c.buffer[n:]
		return n, nil
	}

	bbPool.Put(c.cache)
	c.cache = nil

	if inBufferLen >= n {
		c.inboundBuffer.Discard(n)
		return n, nil
	}
	c.inboundBuffer.Reset()

	remaining := n - inBufferLen
	c.buffer = c.buffer[remaining:]
	return n, nil
}

func (c *conn) Write(p []byte) (n int, err error) {
	if c.isDatagram {
		return len(p), c.sendTo(p)
	}
	return len(p), c.write(p)
}

func (c *conn) Writev(bs [][]byte) (n int, err error) {
	for _, b := range bs {
		n += len(b)
	}
	if c.isDatagram {
		buf := bsPool.Get(n)
		var m int
		for _, b := range bs {
			copy(buf[m:], b)
			m += len(b)
		}
		err = c.sendTo(buf)
		bsPool.Put(buf)
		return
	}
	err = c.writev(bs)
	return
}

func (c *conn) ReadFrom(r io.Reader) (n int64, err error) {
	return c.outboundBuffer.ReadFrom(r)
}

func (c *conn) WriteTo(w io.Writer) (n int64, err error) {
	if c.inboundBuffer.IsEmpty() {
		var m int
		m, err = w.Write(c.buffer)
		c.buffer = c.buffer[m:]
		return int64(m), err
	}
	n, err = c.inboundBuffer.WriteTo(w)
	if err != nil {
		return
	}
	var m int
	m, err = w.Write(c.buffer)
	c.buffer = c.buffer[m:]
	return int64(m), err
}

func (c *conn) Flush() error {
	return c.loop.write(c)
}

func (c *conn) InboundBuffered() int {
	return c.inboundBuffer.Buffered() + len(c.buffer)
}

func (c *conn) OutboundBuffered() int {
	return c.outboundBuffer.Buffered()
}

func (c *conn) SetDeadline(_ time.Time) error {
	return gerrors.ErrUnsupportedOp
}

func (c *conn) SetReadDeadline(_ time.Time) error {
	return gerrors.ErrUnsupportedOp
}

func (c *conn) SetWriteDeadline(_ time.Time) error {
	return gerrors.ErrUnsupportedOp
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }

// ==================================== Concurrency-safe API's ====================================

func (c *conn) AsyncWrite(buf []byte) error {
	if c.isDatagram {
		return c.sendTo(buf)
	}
	return c.loop.poller.Trigger(c.asyncWrite, buf)
}

func (c *conn) AsyncWritev(bs [][]byte) error {
	if c.isDatagram {
		for _, b := range bs {
			if err := c.sendTo(b); err != nil {
				return err
			}
		}
		return nil
	}
	return c.loop.poller.Trigger(c.asyncWritev, bs)
}

func (c *conn) Wake() error {
	return c.loop.poller.UrgentTrigger(func(_ interface{}) error { return c.loop.wake(c) }, nil)
}

func (c *conn) Close() error {
	return c.loop.poller.Trigger(func(_ interface{}) error { return c.loop.closeConn(c, nil) }, nil)
}
