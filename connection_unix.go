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
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/internal/socket"
	"github.com/panjf2000/gnet/listbuffer"
	"github.com/panjf2000/gnet/pool/bytebuffer"
	rbPool "github.com/panjf2000/gnet/pool/ringbuffer"
	"github.com/panjf2000/gnet/ringbuffer"
)

type conn struct {
	fd             int                       // file descriptor
	sa             unix.Sockaddr             // remote socket address
	ctx            interface{}               // user-defined context
	loop           *eventloop                // connected event-loop
	codec          ICodec                    // codec for TCP
	opened         bool                      // connection opened event fired
	localAddr      net.Addr                  // local addr
	remoteAddr     net.Addr                  // remote addr
	byteBuffer     *bytebuffer.ByteBuffer    // bytes buffer for buffering current packet and data in ring-buffer
	inboundBuffer  *ringbuffer.RingBuffer    // buffer for data from the peer
	outboundBuffer listbuffer.ByteBufferList // buffer for data that is ready to write to the peer
	pollAttachment *netpoll.PollAttachment   // connection attachment for poller
}

func newTCPConn(fd int, el *eventloop, sa unix.Sockaddr, codec ICodec, localAddr, remoteAddr net.Addr) (c *conn) {
	c = &conn{
		fd:            fd,
		sa:            sa,
		loop:          el,
		codec:         codec,
		localAddr:     localAddr,
		remoteAddr:    remoteAddr,
		inboundBuffer: rbPool.GetWithSize(ringbuffer.TCPReadBufferSize),
	}
	c.pollAttachment = netpoll.GetPollAttachment()
	c.pollAttachment.FD, c.pollAttachment.Callback = fd, c.handleEvents
	return
}

func (c *conn) releaseTCP() {
	c.opened = false
	c.sa = nil
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
	rbPool.Put(c.inboundBuffer)
	c.inboundBuffer = ringbuffer.EmptyRingBuffer
	c.outboundBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
	netpoll.PutPollAttachment(c.pollAttachment)
}

func newUDPConn(fd int, el *eventloop, localAddr net.Addr, sa unix.Sockaddr) *conn {
	return &conn{
		fd:         fd,
		sa:         sa,
		loop:       el,
		localAddr:  localAddr,
		remoteAddr: socket.SockaddrToUDPAddr(sa),
	}
}

func (c *conn) releaseUDP() {
	c.ctx = nil
	c.localAddr = nil
	c.remoteAddr = nil
}

func (c *conn) open(buf []byte) error {
	defer c.loop.eventHandler.AfterWrite(c, buf)

	c.loop.eventHandler.PreWrite(c)
	n, err := unix.Write(c.fd, buf)
	if err != nil && err == unix.EAGAIN {
		c.outboundBuffer.PushBytes(buf)
		return nil
	}

	if err == nil && n < len(buf) {
		c.outboundBuffer.PushBytes(buf[n:])
	}

	return err
}

func (c *conn) read() ([]byte, error) {
	return c.codec.Decode(c)
}

func (c *conn) write(buf []byte) (err error) {
	defer c.loop.eventHandler.AfterWrite(c, buf)

	var packet []byte
	if packet, err = c.codec.Encode(c, buf); err != nil {
		return
	}

	c.loop.eventHandler.PreWrite(c)

	// If there is pending data in outbound buffer, the current data ought to be appended to the outbound buffer
	// for maintaining the sequence of network packets.
	if !c.outboundBuffer.IsEmpty() {
		c.outboundBuffer.PushBytes(packet)
		return
	}

	var n int
	if n, err = unix.Write(c.fd, packet); err != nil {
		// A temporary error occurs, append the data to outbound buffer, writing it back to the peer in the next round.
		if err == unix.EAGAIN {
			c.outboundBuffer.PushBytes(packet)
			err = c.loop.poller.ModReadWrite(c.pollAttachment)
			return
		}
		return c.loop.loopCloseConn(c, os.NewSyscallError("write", err))
	}
	// Failed to send all data back to the peer, buffer the leftover data for the next round.
	if n < len(packet) {
		c.outboundBuffer.PushBytes(packet[n:])
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

func (c *conn) sendTo(buf []byte) error {
	c.loop.eventHandler.PreWrite(c)
	defer c.loop.eventHandler.AfterWrite(c, buf)
	return unix.Sendto(c.fd, buf, 0, c.sa)
}

// ================================= Public APIs of gnet.Conn =================================

func (c *conn) Read() []byte {
	buf, _ := c.inboundBuffer.PeekAll()
	return buf
}

func (c *conn) ResetBuffer() {
	c.inboundBuffer.Reset()
}

func (c *conn) ReadN(n int) (int, []byte) {
	inBufferLen := c.inboundBuffer.Length()
	if inBufferLen < n || n <= 0 {
		buf, _ := c.inboundBuffer.PeekAll()
		return inBufferLen, buf
	}
	buf, _ := c.inboundBuffer.Peek(n)
	return n, buf
}

func (c *conn) ShiftN(n int) int {
	c.inboundBuffer.Discard(n)
	return n
}

func (c *conn) BufferLength() int {
	return c.inboundBuffer.Length()
}

func (c *conn) AsyncWrite(buf []byte) error {
	return c.loop.poller.Trigger(c.asyncWrite, buf)
}

func (c *conn) SendTo(buf []byte) error {
	return c.sendTo(buf)
}

func (c *conn) Wake() error {
	return c.loop.poller.UrgentTrigger(func(_ interface{}) error { return c.loop.loopWake(c) }, nil)
}

func (c *conn) Close() error {
	return c.loop.poller.Trigger(func(_ interface{}) error { return c.loop.loopCloseConn(c, nil) }, nil)
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
