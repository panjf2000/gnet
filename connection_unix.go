// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/gfd"
	"github.com/panjf2000/gnet/v2/pkg/bs"
	"github.com/panjf2000/gnet/v2/pkg/buffer/elastic"
	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	gio "github.com/panjf2000/gnet/v2/pkg/io"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"github.com/panjf2000/gnet/v2/pkg/socket"
)

type conn struct {
	fd             int                    // file descriptor
	gfd            gfd.GFD                // gnet file descriptor
	ctx            any                    // user-defined context
	remote         unix.Sockaddr          // remote socket address
	proto          string                 // protocol name: "tcp", "udp", or "unix".
	localAddr      net.Addr               // local addr
	remoteAddr     net.Addr               // remote addr
	loop           *eventloop             // connected event-loop
	outboundBuffer elastic.Buffer         // buffer for data that is eligible to be sent to the remote
	pollAttachment netpoll.PollAttachment // connection attachment for poller
	inboundBuffer  elastic.RingBuffer     // buffer for leftover data from the remote
	buffer         []byte                 // buffer for the latest bytes
	cache          []byte                 // temporary cache for the inbound data
	isDatagram     bool                   // UDP protocol
	opened         bool                   // connection opened event fired
	isEOF          bool                   // whether the connection has reached EOF
}

func newStreamConn(proto string, fd int, el *eventloop, sa unix.Sockaddr, localAddr, remoteAddr net.Addr) (c *conn) {
	c = &conn{
		fd:             fd,
		proto:          proto,
		remote:         sa,
		loop:           el,
		localAddr:      localAddr,
		remoteAddr:     remoteAddr,
		pollAttachment: netpoll.PollAttachment{FD: fd},
	}
	c.pollAttachment.Callback = c.processIO
	c.outboundBuffer.Reset(el.engine.opts.WriteBufferCap)
	return
}

func newUDPConn(fd int, el *eventloop, localAddr net.Addr, sa unix.Sockaddr, connected bool) (c *conn) {
	c = &conn{
		fd:             fd,
		proto:          "udp",
		gfd:            gfd.NewGFD(fd, el.idx, 0, 0),
		remote:         sa,
		loop:           el,
		localAddr:      localAddr,
		remoteAddr:     socket.SockaddrToUDPAddr(sa),
		isDatagram:     true,
		pollAttachment: netpoll.PollAttachment{FD: fd, Callback: el.readUDP},
	}
	if connected {
		c.remote = nil
	}
	return
}

func (c *conn) release() {
	c.opened = false
	c.isEOF = false
	c.ctx = nil
	c.buffer = nil
	if addr, ok := c.localAddr.(*net.TCPAddr); ok && len(c.loop.listeners) == 0 && len(addr.Zone) > 0 {
		bsPool.Put(bs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.remoteAddr.(*net.TCPAddr); ok && len(addr.Zone) > 0 {
		bsPool.Put(bs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.localAddr.(*net.UDPAddr); ok && len(c.loop.listeners) == 0 && len(addr.Zone) > 0 {
		bsPool.Put(bs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.remoteAddr.(*net.UDPAddr); ok && len(addr.Zone) > 0 {
		bsPool.Put(bs.StringToBytes(addr.Zone))
	}
	c.localAddr = nil
	c.remoteAddr = nil
	if !c.isDatagram {
		c.remote = nil
		c.inboundBuffer.Done()
		c.outboundBuffer.Release()
	}
}

func (c *conn) open(buf []byte) error {
	if c.isDatagram && c.remote == nil {
		return unix.Send(c.fd, buf, 0)
	}

	for {
		n, err := unix.Write(c.fd, buf)
		if err != nil {
			if err == unix.EAGAIN {
				_, _ = c.outboundBuffer.Write(buf)
				break
			}
			return err
		}
		buf = buf[n:]
		if len(buf) == 0 {
			break
		}
	}

	return nil
}

func (c *conn) write(data []byte) (n int, err error) {
	isET := c.loop.engine.opts.EdgeTriggeredIO
	n = len(data)
	// If there is pending data in outbound buffer,
	// the current data ought to be appended to the
	// outbound buffer for maintaining the sequence
	// of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(data)
		return
	}

	defer func() {
		if err != nil {
			_ = c.loop.close(c, os.NewSyscallError("write", err))
		}
	}()

	var sent int
loop:
	if sent, err = unix.Write(c.fd, data); err != nil {
		// A temporary error occurs, append the data to outbound buffer,
		// writing it back to the remote in the next round for LT mode.
		if err == unix.EAGAIN {
			_, err = c.outboundBuffer.Write(data)
			if !isET {
				err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
			}
			return
		}
		return 0, err
	}
	data = data[sent:]
	if isET && len(data) > 0 {
		goto loop
	}
	// Failed to send all data back to the remote, buffer the leftover data for the next round.
	if len(data) > 0 {
		_, _ = c.outboundBuffer.Write(data)
		err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
	}

	return
}

func (c *conn) writev(bs [][]byte) (n int, err error) {
	isET := c.loop.engine.opts.EdgeTriggeredIO

	for _, b := range bs {
		n += len(b)
	}

	// If there is pending data in outbound buffer,
	// the current data ought to be appended to the
	// outbound buffer for maintaining the sequence
	// of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Writev(bs)
		return
	}

	defer func() {
		if err != nil {
			_ = c.loop.close(c, os.NewSyscallError("writev", err))
		}
	}()

	remaining := n
	var sent int
loop:
	if sent, err = gio.Writev(c.fd, bs); err != nil {
		// A temporary error occurs, append the data to outbound buffer,
		// writing it back to the remote in the next round for LT mode.
		if err == unix.EAGAIN {
			_, err = c.outboundBuffer.Writev(bs)
			if !isET {
				err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
			}
			return
		}
		return 0, err
	}
	pos := len(bs)
	if remaining -= sent; remaining > 0 {
		for i := range bs {
			bn := len(bs[i])
			if sent < bn {
				bs[i] = bs[i][sent:]
				pos = i
				break
			}
			sent -= bn
		}
	}
	bs = bs[pos:]
	if isET && remaining > 0 {
		goto loop
	}

	// Failed to send all data back to the remote, buffer the leftover data for the next round.
	if remaining > 0 {
		_, _ = c.outboundBuffer.Writev(bs)
		err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
	}

	return
}

type asyncWriteHook struct {
	callback AsyncCallback
	data     []byte
}

func (c *conn) asyncWrite(a any) (err error) {
	hook := a.(*asyncWriteHook)
	defer func() {
		if hook.callback != nil {
			_ = hook.callback(c, err)
		}
	}()

	if !c.opened {
		return net.ErrClosed
	}

	_, err = c.write(hook.data)
	return
}

type asyncWritevHook struct {
	callback AsyncCallback
	data     [][]byte
}

func (c *conn) asyncWritev(a any) (err error) {
	hook := a.(*asyncWritevHook)
	defer func() {
		if hook.callback != nil {
			_ = hook.callback(c, err)
		}
	}()

	if !c.opened {
		return net.ErrClosed
	}

	_, err = c.writev(hook.data)
	return
}

func (c *conn) sendTo(buf []byte, addr unix.Sockaddr) (n int, err error) {
	defer func() {
		if err != nil {
			n = 0
		}
	}()

	if addr != nil {
		return len(buf), unix.Sendto(c.fd, buf, 0, addr)
	}
	if c.remote == nil { // connected UDP socket of client
		return len(buf), unix.Send(c.fd, buf, 0)
	}
	return len(buf), unix.Sendto(c.fd, buf, 0, c.remote) // unconnected UDP socket of server
}

func (c *conn) resetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
	c.inboundBuffer.Done()
}

func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		if n == 0 && len(p) > 0 {
			err = io.ErrShortBuffer
		}
		return
	}
	n, _ = c.inboundBuffer.Read(p)
	if n == len(p) {
		return
	}
	m := copy(p[n:], c.buffer)
	n += m
	c.buffer = c.buffer[m:]
	return
}

func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer[:n]
		c.buffer = c.buffer[n:]
		return
	}

	buf = bsPool.Get(n)
	_, err = c.Read(buf)
	return
}

func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		return c.buffer[:n], err
	}

	head, tail := c.inboundBuffer.Peek(n)
	if len(head) == n {
		return head, err
	}
	buf = bsPool.Get(n)[:0]
	buf = append(buf, head...)
	buf = append(buf, tail...)
	if inBufferLen >= n {
		return
	}

	remaining := n - inBufferLen
	buf = append(buf, c.buffer[:remaining]...)
	c.cache = buf
	return
}

func (c *conn) Discard(n int) (int, error) {
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n >= totalLen || n <= 0 {
		c.resetBuffer()
		return totalLen, nil
	}

	if c.inboundBuffer.IsEmpty() {
		c.buffer = c.buffer[n:]
		return n, nil
	}

	discarded, _ := c.inboundBuffer.Discard(n)
	if discarded < inBufferLen {
		return discarded, nil
	}

	remaining := n - inBufferLen
	c.buffer = c.buffer[remaining:]
	return n, nil
}

func (c *conn) Write(p []byte) (int, error) {
	if c.isDatagram {
		return c.sendTo(p, nil)
	}
	return c.write(p)
}

func (c *conn) SendTo(p []byte, addr net.Addr) (int, error) {
	if !c.isDatagram {
		return 0, errorx.ErrUnsupportedOp
	}

	sa := socket.NetAddrToSockaddr(addr)
	if sa == nil {
		return 0, errorx.ErrInvalidNetworkAddress
	}

	return c.sendTo(p, sa)
}

func (c *conn) Writev(bs [][]byte) (int, error) {
	if c.isDatagram {
		return 0, errorx.ErrUnsupportedOp
	}
	return c.writev(bs)
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	return c.outboundBuffer.ReadFrom(r)
}

func (c *conn) WriteTo(w io.Writer) (n int64, err error) {
	if !c.inboundBuffer.IsEmpty() {
		if n, err = c.inboundBuffer.WriteTo(w); err != nil {
			return
		}
	}
	var m int
	m, err = w.Write(c.buffer)
	n += int64(m)
	c.buffer = c.buffer[m:]
	return
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

func (c *conn) Context() any         { return c.ctx }
func (c *conn) SetContext(ctx any)   { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr  { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr { return c.remoteAddr }

// Implementation of Socket interface

// func (c *conn) Gfd() gfd.GFD             { return c.gfd }

func (c *conn) Fd() int                        { return c.fd }
func (c *conn) Dup() (fd int, err error)       { return socket.Dup(c.fd) }
func (c *conn) SetReadBuffer(bytes int) error  { return socket.SetRecvBuffer(c.fd, bytes) }
func (c *conn) SetWriteBuffer(bytes int) error { return socket.SetSendBuffer(c.fd, bytes) }
func (c *conn) SetLinger(sec int) error        { return socket.SetLinger(c.fd, sec) }
func (c *conn) SetNoDelay(noDelay bool) error {
	return socket.SetNoDelay(c.fd, func(b bool) int {
		if b {
			return 1
		}
		return 0
	}(noDelay))
}

func (c *conn) SetKeepAlivePeriod(d time.Duration) error {
	if c.proto != "tcp" {
		return errorx.ErrUnsupportedOp
	}
	return socket.SetKeepAlivePeriod(c.fd, int(d.Seconds()))
}

func (c *conn) SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error {
	if c.proto != "tcp" {
		return errorx.ErrUnsupportedOp
	}
	return socket.SetKeepAlive(c.fd, enabled, int(idle.Seconds()), int(intvl.Seconds()), cnt)
}

func (c *conn) AsyncWrite(buf []byte, callback AsyncCallback) error {
	if c.isDatagram {
		_, err := c.sendTo(buf, nil)
		// TODO: it will not go asynchronously with UDP, so calling a callback is needless,
		//  we may remove this branch in the future, please don't rely on the callback
		// 	to do something important under UDP, if you're working with UDP, just call Conn.Write
		// 	to send back your data.
		if callback != nil {
			_ = callback(nil, nil)
		}
		return err
	}
	return c.loop.poller.Trigger(queue.HighPriority, c.asyncWrite, &asyncWriteHook{callback, buf})
}

func (c *conn) AsyncWritev(bs [][]byte, callback AsyncCallback) error {
	if c.isDatagram {
		return errorx.ErrUnsupportedOp
	}
	return c.loop.poller.Trigger(queue.HighPriority, c.asyncWritev, &asyncWritevHook{callback, bs})
}

func (c *conn) Wake(callback AsyncCallback) error {
	return c.loop.poller.Trigger(queue.LowPriority, func(_ any) (err error) {
		err = c.loop.wake(c)
		if callback != nil {
			_ = callback(c, err)
		}
		return
	}, nil)
}

func (c *conn) CloseWithCallback(callback AsyncCallback) error {
	return c.loop.poller.Trigger(queue.LowPriority, func(_ any) (err error) {
		err = c.loop.close(c, nil)
		if callback != nil {
			_ = callback(c, err)
		}
		return
	}, nil)
}

func (c *conn) Close() error {
	return c.loop.poller.Trigger(queue.LowPriority, func(_ any) (err error) {
		err = c.loop.close(c, nil)
		return
	}, nil)
}

func (c *conn) EventLoop() EventLoop {
	return c.loop
}

func (*conn) SetDeadline(_ time.Time) error {
	return errorx.ErrUnsupportedOp
}

func (*conn) SetReadDeadline(_ time.Time) error {
	return errorx.ErrUnsupportedOp
}

func (*conn) SetWriteDeadline(_ time.Time) error {
	return errorx.ErrUnsupportedOp
}
