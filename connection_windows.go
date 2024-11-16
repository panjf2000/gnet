// Copyright (c) 2023 The Gnet Authors. All rights reserved.
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

package gnet

import (
	"errors"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/panjf2000/ants/v2"
	"golang.org/x/sys/windows"

	"github.com/panjf2000/gnet/v2/pkg/buffer/elastic"
	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
	bsPool "github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type netErr struct {
	c   *conn
	err error
}

type tcpConn struct {
	c *conn
	b *bbPool.ByteBuffer
}

type udpConn struct {
	c *conn
}

type openConn struct {
	c  *conn
	cb func()
}

type conn struct {
	pc            net.PacketConn
	ctx           any                // user-defined context
	loop          *eventloop         // owner event-loop
	buffer        *bbPool.ByteBuffer // reuse memory of inbound data as a temporary buffer
	cache         []byte             // temporary cache for the inbound data
	rawConn       net.Conn           // original connection
	localAddr     net.Addr           // local server addr
	remoteAddr    net.Addr           // remote addr
	inboundBuffer elastic.RingBuffer // buffer for data from the remote
}

func packTCPConn(c *conn, buf []byte) *tcpConn {
	b := bbPool.Get()
	_, _ = b.Write(buf)
	return &tcpConn{c: c, b: b}
}

func unpackTCPConn(tc *tcpConn) *conn {
	if tc.c.buffer == nil { // the connection has been closed
		return nil
	}
	_, _ = tc.c.buffer.Write(tc.b.B)
	bbPool.Put(tc.b)
	tc.b = nil
	return tc.c
}

func packUDPConn(c *conn, buf []byte) *udpConn {
	_, _ = c.buffer.Write(buf)
	return &udpConn{c}
}

func newTCPConn(nc net.Conn, el *eventloop) (c *conn) {
	return &conn{
		loop:       el,
		buffer:     bbPool.Get(),
		rawConn:    nc,
		localAddr:  nc.LocalAddr(),
		remoteAddr: nc.RemoteAddr(),
	}
}

func (c *conn) release() {
	c.ctx = nil
	c.localAddr = nil
	if c.rawConn != nil {
		c.rawConn = nil
		c.remoteAddr = nil
	}
	c.inboundBuffer.Done()
	bbPool.Put(c.buffer)
	c.buffer = nil
}

func newUDPConn(el *eventloop, pc net.PacketConn, localAddr, remoteAddr net.Addr) *conn {
	return &conn{
		pc:         pc,
		loop:       el,
		buffer:     bbPool.Get(),
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

func (c *conn) resetBuffer() {
	c.buffer.Reset()
	c.inboundBuffer.Reset()
	c.inboundBuffer.Done()
}

func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer.B)
		c.buffer.B = c.buffer.B[n:]
		if n == 0 && len(p) > 0 {
			err = io.ErrShortBuffer
		}
		return
	}
	n, _ = c.inboundBuffer.Read(p)
	if n == len(p) {
		return
	}
	m := copy(p[n:], c.buffer.B)
	n += m
	c.buffer.B = c.buffer.B[m:]
	return
}

func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer.B[:n]
		c.buffer.B = c.buffer.B[n:]
		return
	}

	buf = bsPool.Get(n)
	_, err = c.Read(buf)
	return
}

func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		return c.buffer.B[:n], err
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
	buf = append(buf, c.buffer.B[:remaining]...)
	c.cache = buf
	return
}

func (c *conn) Discard(n int) (int, error) {
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n >= totalLen || n <= 0 {
		c.resetBuffer()
		return totalLen, nil
	}

	if c.inboundBuffer.IsEmpty() {
		c.buffer.B = c.buffer.B[n:]
		return n, nil
	}

	discarded, _ := c.inboundBuffer.Discard(n)
	if discarded < inBufferLen {
		return discarded, nil
	}

	remaining := n - inBufferLen
	c.buffer.B = c.buffer.B[remaining:]
	return n, nil
}

func (c *conn) Write(p []byte) (int, error) {
	if c.rawConn == nil && c.pc == nil {
		return 0, net.ErrClosed
	}
	if c.rawConn != nil {
		return c.rawConn.Write(p)
	}
	return c.pc.WriteTo(p, c.remoteAddr)
}

func (c *conn) Writev(bs [][]byte) (int, error) {
	if c.rawConn != nil {
		bb := bbPool.Get()
		defer bbPool.Put(bb)
		for i := range bs {
			_, _ = bb.Write(bs[i])
		}
		return c.rawConn.Write(bb.Bytes())
	}
	return 0, net.ErrClosed
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	if c.rawConn != nil {
		return io.Copy(c.rawConn, r)
	}
	return 0, net.ErrClosed
}

func (c *conn) WriteTo(w io.Writer) (n int64, err error) {
	if !c.inboundBuffer.IsEmpty() {
		if n, err = c.inboundBuffer.WriteTo(w); err != nil {
			return
		}
	}

	if c.buffer == nil {
		return 0, nil
	}
	defer c.buffer.Reset()
	return c.buffer.WriteTo(w)
}

func (c *conn) Flush() error {
	return nil
}

func (c *conn) InboundBuffered() int {
	if c.buffer == nil {
		return 0
	}
	return c.inboundBuffer.Buffered() + c.buffer.Len()
}

func (c *conn) OutboundBuffered() int {
	return 0
}

func (c *conn) Context() any         { return c.ctx }
func (c *conn) SetContext(ctx any)   { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr  { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *conn) Fd() (fd int) {
	if c.rawConn == nil {
		return -1
	}

	rc, err := c.rawConn.(syscall.Conn).SyscallConn()
	if err != nil {
		return -1
	}
	if err := rc.Control(func(i uintptr) {
		fd = int(i)
	}); err != nil {
		return -1
	}
	return
}

func (c *conn) Dup() (fd int, err error) {
	if c.rawConn == nil && c.pc == nil {
		return -1, net.ErrClosed
	}

	var (
		sc syscall.Conn
		ok bool
	)
	if c.rawConn != nil {
		sc, ok = c.rawConn.(syscall.Conn)
	} else {
		sc, ok = c.pc.(syscall.Conn)
	}

	if !ok {
		return -1, errors.New("failed to convert net.Conn to syscall.Conn")
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		return -1, errors.New("failed to get syscall.RawConn from net.Conn")
	}

	var dupHandle windows.Handle
	e := rc.Control(func(fd uintptr) {
		process := windows.CurrentProcess()
		err = windows.DuplicateHandle(
			process,
			windows.Handle(fd),
			process,
			&dupHandle,
			0,
			true,
			windows.DUPLICATE_SAME_ACCESS,
		)
	})
	if err != nil {
		return -1, err
	}
	if e != nil {
		return -1, e
	}

	return int(dupHandle), nil
}

func (c *conn) SetReadBuffer(bytes int) error {
	if c.rawConn == nil && c.pc == nil {
		return net.ErrClosed
	}

	if c.rawConn != nil {
		return c.rawConn.(interface{ SetReadBuffer(int) error }).SetReadBuffer(bytes)
	}
	return c.pc.(interface{ SetReadBuffer(int) error }).SetReadBuffer(bytes)
}

func (c *conn) SetWriteBuffer(bytes int) error {
	if c.rawConn == nil && c.pc == nil {
		return net.ErrClosed
	}
	if c.rawConn != nil {
		return c.rawConn.(interface{ SetWriteBuffer(int) error }).SetWriteBuffer(bytes)
	}
	return c.pc.(interface{ SetWriteBuffer(int) error }).SetWriteBuffer(bytes)
}

func (c *conn) SetLinger(sec int) error {
	if c.rawConn == nil {
		return net.ErrClosed
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok {
		return errorx.ErrUnsupportedOp
	}
	return tc.SetLinger(sec)
}

func (c *conn) SetNoDelay(noDelay bool) error {
	if c.rawConn == nil {
		return net.ErrClosed
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok {
		return errorx.ErrUnsupportedOp
	}
	return tc.SetNoDelay(noDelay)
}

func (c *conn) SetKeepAlivePeriod(d time.Duration) error {
	if c.rawConn == nil {
		return net.ErrClosed
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok || d < 0 {
		return errorx.ErrUnsupportedOp
	}
	if err := tc.SetKeepAlive(true); err != nil {
		return err
	}
	if err := tc.SetKeepAlivePeriod(d); err != nil {
		_ = tc.SetKeepAlive(false)
		return err
	}

	return nil
}

type nonBlockingPool struct {
	*goPool.Pool
}

func (np *nonBlockingPool) Go(task func()) (err error) {
	if err = np.Submit(task); err == ants.ErrPoolOverload {
		go task()
	}
	return
}

var workerPool = nonBlockingPool{Pool: goPool.Default()}

// Gfd return an uninitialized GFD which is not valid,
// this method is only implemented for compatibility, don't use it on Windows.
// func (c *conn) Gfd() gfd.GFD { return gfd.GFD{} }

func (c *conn) AsyncWrite(buf []byte, cb AsyncCallback) error {
	_, err := c.Write(buf)

	callback := func() error {
		if cb != nil {
			_ = cb(c, err)
		}
		return err
	}

	select {
	case c.loop.ch <- callback:
	default:
		// If the event-loop channel is full, asynchronize this operation to avoid blocking the eventloop.
		err = workerPool.Go(func() {
			c.loop.ch <- callback
		})
	}

	return err
}

func (c *conn) AsyncWritev(bs [][]byte, cb AsyncCallback) error {
	buf := bbPool.Get()
	for _, b := range bs {
		_, _ = buf.Write(b)
	}
	return c.AsyncWrite(buf.Bytes(), func(c Conn, err error) error {
		defer bbPool.Put(buf)
		if cb == nil {
			return err
		}
		return cb(c, err)
	})
}

func (c *conn) Wake(cb AsyncCallback) (err error) {
	wakeFn := func() (err error) {
		err = c.loop.wake(c)
		if cb != nil {
			_ = cb(c, err)
		}
		return
	}

	select {
	case c.loop.ch <- wakeFn:
	default:
		// If the event-loop channel is full, asynchronize this operation to avoid blocking the eventloop.
		err = workerPool.Go(func() {
			c.loop.ch <- wakeFn
		})
	}

	return
}

func (c *conn) Close() (err error) {
	closeFn := func() error {
		return c.loop.close(c, nil)
	}

	select {
	case c.loop.ch <- closeFn:
	default:
		// If the event-loop channel is full, asynchronize this operation to avoid blocking the eventloop.
		err = workerPool.Go(func() {
			c.loop.ch <- closeFn
		})
	}

	return
}

func (c *conn) CloseWithCallback(cb AsyncCallback) (err error) {
	closeFn := func() (err error) {
		err = c.loop.close(c, nil)
		if cb != nil {
			_ = cb(c, err)
		}
		return
	}

	select {
	case c.loop.ch <- closeFn:
	default:
		// If the event-loop channel is full, asynchronize this operation to avoid blocking the eventloop.
		err = workerPool.Go(func() {
			c.loop.ch <- closeFn
		})
	}

	return
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
