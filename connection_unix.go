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
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/nelthaarion/gnet/v2/internal/gfd"
	"github.com/nelthaarion/gnet/v2/pkg/buffer/elastic"
	errorx "github.com/nelthaarion/gnet/v2/pkg/errors"
	gio "github.com/nelthaarion/gnet/v2/pkg/io"
	"github.com/nelthaarion/gnet/v2/pkg/netpoll"
	bsPool "github.com/nelthaarion/gnet/v2/pkg/pool/byteslice"
	"github.com/nelthaarion/gnet/v2/pkg/queue"
	"github.com/nelthaarion/gnet/v2/pkg/socket"
)

type conn struct {
	fd             int                    // file descriptor
	gfd            gfd.GFD                // gnet file descriptor
	ctx            any                    // user-defined context
	safeCtx        atomic.Pointer[any]    // safe user-defined context
	remote         unix.Sockaddr          // remote socket address
	proto          string                 // protocol name: "tcp", "udp", or "unix".
	localAddr      net.Addr               // local addr
	remoteAddr     net.Addr               // remote addr
	loop           *eventloop             // connected event-loop
	outboundBuffer elastic.Buffer         // buffer for data that is eligible to be sent to the remote
	pollAttachment netpoll.PollAttachment // connection attachment for poller
	inboundBuffer  elastic.RingBuffer     // buffer for leftover data from the remote
	buffer         []byte                 // buffer for the latest bytes
	cache          []byte                 // temporary cache for the inbound data (owned by bsPool, freed by Discard/release)
	peekCaches     [][]byte               // allocated Peek results, valid until Discard/release
	isDatagram     bool                   // UDP protocol
	opened         bool                   // connection opened event fired
	isEOF          bool                   // whether the connection has reached EOF
	cork           corkState              // write-coalescing state of the OnTraffic callback that is running
	corkDirty      bool                   // writes were buffered by coalescing, flush when the callback returns
	corkBatch      bool                   // the previous callback wrote more than once: hold back all writes of the next one
	corkN          uint32                 // writes made since the coalescing window of the current callback opened
}

// corkState is the write-coalescing state of the OnTraffic (or Wake) callback that is
// currently running on a connection.
//
// A callback that handles several requests per read - HTTP pipelining, Redis-style
// batches, framed RPC, WebSocket bursts - typically answers each of them with its own
// Write, and every Write is a write(2). The first Write of a callback still goes
// straight to the kernel, so the common one-request-one-response exchange pays
// nothing extra and its latency is untouched; the Writes that follow it in the same
// callback are appended to the outbound buffer instead and leave in one write(2)
// when the callback returns. N pipelined responses cost two write(2) calls, not N.
//
// A connection whose previous callback wrote more than once is assumed to be
// pipelining and its next callback holds back even the first Write (as long as it is
// small, see corkHoldMax): all N responses then leave in a single write(2) when the
// callback returns. One extra write(2) per batch is the largest single cost of the
// pipelined case, as much as sending the other N-1 responses together. A callback
// that turns out to write only once is not hurt by the guess: its Write is merely
// sent when the callback returns instead of inside it, and the next callback is back
// to the direct first Write.
//
// Coalescing only exists in level-triggered mode. In edge-triggered mode the event
// loop must drain the socket before it returns, so every write stays immediate.
type corkState uint8

const (
	corkOff   corkState = iota // no callback is running (or ET mode): writes go straight to the kernel
	corkOpen                   // a callback is running and has not written yet
	corkHold                   // like corkOpen, but the connection pipelines: even the first (small) write is buffered
	corkArmed                  // the callback has made its first write: the next ones are buffered
)

// corkHoldMax is the largest first Write that corkHold buffers. Copying a bigger
// one into the outbound buffer costs more than the write(2) it could save, so it is
// sent directly, as in corkOpen.
const corkHoldMax = 4 << 10

// openCork opens the write-coalescing window of an OnTraffic (or Wake) callback.
func (c *conn) openCork() {
	c.corkN = 0
	if c.corkBatch {
		c.cork = corkHold
	} else {
		c.cork = corkOpen
	}
}

// closeCork closes the window that openCork opened (opened tells whether there was
// one) and reports whether the callback left writes in the outbound buffer that have
// to be flushed now. It also remembers whether the callback wrote more than once,
// which decides how the next one starts.
func (c *conn) closeCork(opened bool) (flush bool) {
	flush = c.corkDirty
	if opened {
		c.corkBatch = c.corkN > 1
	}
	c.cork, c.corkDirty = corkOff, false
	return
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
		gfd:            gfd.NewGFDWithSeq(fd, el.idx, 0, 0, &el.gfdSeq),
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
	c.releasePeekCaches()
	c.opened = false
	c.isEOF = false
	c.cork, c.corkDirty, c.corkBatch, c.corkN = corkOff, false, false, 0
	c.ctx = nil
	c.safeCtx.Store(nil)
	c.buffer = nil

	// FIX P-8: If the caller invoked Peek() but never called Discard() (e.g. the
	// connection was closed mid-protocol), c.cache still holds a pooled slice that
	// was never returned. Return it now to prevent a permanent pool leak.
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	// FIX S-2: The previous code called bsPool.Put(bs.StringToBytes(addr.Zone))
	// for TCPAddr and UDPAddr Zone fields. bs.StringToBytes returns an unsafe slice
	// pointing into the Go string's backing array, which is owned by the net package
	// runtime — NOT by bsPool. Returning foreign memory to the pool means the pool
	// may hand it to another connection while the original string is still alive,
	// creating a Use-After-Free / data-corruption hazard on IPv6 link-local
	// connections (e.g. "fe80::1%eth0").
	//
	// The net package manages Zone strings with the GC; we must not touch them.
	// Simply nil out the address pointers and let the GC do its job.
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
		n, err := gio.WriteSocketNB(c.fd, buf)
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
	c.corkN++
	// If there is pending data in outbound buffer,
	// the current data ought to be appended to the
	// outbound buffer for maintaining the sequence
	// of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(data)
		return
	}

	// Write coalescing, see corkState: the first Write of an OnTraffic callback is sent
	// right away (unless the connection pipelines), every later one is buffered and
	// flushed once the callback returns.
	if c.cork != corkOff {
		if c.cork == corkArmed || (c.cork == corkHold && n <= corkHoldMax) {
			c.cork = corkArmed
			_, _ = c.outboundBuffer.Write(data)
			c.corkDirty = true
			return
		}
		c.cork = corkArmed
	}

	// The old body opened this function with a `defer func(){ if err != nil {
	// c.loop.close(...) } }()`. The closure made the write path non-inlinable
	// and forced the error case through an indirection; the error paths below
	// now close the connection inline, so the fast path is closure-free.

	var sent int
loop:
	if sent, err = gio.WriteSocketNB(c.fd, data); err != nil {
		// A temporary error occurs, append the data to outbound buffer,
		// writing it back to the remote in the next round for LT mode.
		if err == unix.EAGAIN {
			_, err = c.outboundBuffer.Write(data)
			if !isET {
				err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
			}
			if err != nil {
				// The deferred handler would have done this; the point of
				// inlining it is to keep the closure off the hot path, so the
				// error path has to do its own cleanup.
				_ = c.loop.close(c, os.NewSyscallError("write", err))
			}
			return
		}
		// Not a temporary error: the connection is gone, exactly as the old
		// deferred handler would have reported it.
		_ = c.loop.close(c, os.NewSyscallError("write", err))
		return 0, err
	}
	data = data[sent:]
	// FIX P-4: In ET mode the original code looped unconditionally via
	// "goto loop" whenever data remained. If the kernel's send-buffer is
	// congested (small cwnd) this could spin forever inside Write(), blocking
	// the event-loop goroutine and starving all other connections.
	//
	// Guard: only retry immediately if EAGAIN was NOT returned and the kernel
	// accepted a non-zero amount this iteration (i.e. progress was made).
	// When no progress can be made, fall through to the outboundBuffer path
	// exactly like eventloop.write() does with its chunk-based threshold.
	if isET && len(data) > 0 && sent > 0 {
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
	c.corkN++

	// If there is pending data in outbound buffer,
	// the current data ought to be appended to the
	// outbound buffer for maintaining the sequence
	// of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Writev(bs)
		return
	}

	// Write coalescing, see corkState and conn.write.
	if c.cork != corkOff {
		if c.cork == corkArmed || (c.cork == corkHold && n <= corkHoldMax) {
			c.cork = corkArmed
			_, _ = c.outboundBuffer.Writev(bs)
			c.corkDirty = true
			return
		}
		c.cork = corkArmed
	}

	// FIX M-2: writev(2) rejects an iovec list longer than IOV_MAX (1024 on Linux
	// and the BSDs) with EINVAL, and this method treats every error other than
	// EAGAIN as fatal — the deferred handler below closes the connection over it. So
	// a caller passing more than 1024 buffers lost its connection to a call that is
	// perfectly legal at the API level. Only the first IOV_MAX entries are handed to
	// the kernel; the remainder is carried in overflow and appended to the outbound
	// buffer with whatever else could not be sent in this round.
	var overflow [][]byte
	if len(bs) > iovMax {
		overflow = bs[iovMax:]
		bs = bs[:iovMax]
	}

	defer func() {
		if err != nil {
			_ = c.loop.close(c, os.NewSyscallError("writev", err))
		}
	}()

	// remaining counts every byte handed to this method, including the overflow that
	// is never attempted, so the tail of the function buffers exactly what is left.
	remaining := n
	var sent int
loop:
	if sent, err = gio.Writev(c.fd, bs); err != nil {
		// A temporary error occurs, append the data to outbound buffer,
		// writing it back to the remote in the next round for LT mode.
		if err == unix.EAGAIN {
			_, err = c.outboundBuffer.Writev(bs)
			if len(overflow) > 0 {
				if _, werr := c.outboundBuffer.Writev(overflow); err == nil {
					err = werr
				}
			}
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
	// FIX P-4 (writev): Same guard as write() above — only loop again if
	// progress was made, preventing an infinite spin under congestion. Only loop
	// when there is still something to send: once the kernel has taken every
	// iovec, bs is empty and another writev would be a no-op that reports no
	// progress.
	if isET && remaining > 0 && sent > 0 && len(bs) > 0 {
		goto loop
	}

	// Failed to send all data back to the remote, buffer the leftover data for the next round.
	if remaining > 0 {
		_, _ = c.outboundBuffer.Writev(bs)
		if len(overflow) > 0 {
			_, _ = c.outboundBuffer.Writev(overflow)
		}
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

// Next returns the next n bytes and advances the inbound buffer.
//
// FIX P-2: When inboundBuffer is non-empty, Next() allocated from bsPool
// (via bsPool.Get) but never called bsPool.Put — leaking the pooled slice
// on every fragmented read. The fix mirrors the Peek()/Discard() pattern:
// the allocated slice is stored in c.cache so that the next call to Next()
// or Discard() can return it to the pool. release() also returns any
// outstanding c.cache slice (see FIX P-8 in release()).
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

	// Free any previous pool slice before allocating a new one.
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	buf = bsPool.Get(n)
	_, err = c.Read(buf)
	// Track the pooled slice so Discard() or release() can return it.
	c.cache = buf
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
	if inBufferLen < n {
		buf = append(buf, c.buffer[:n-inBufferLen]...)
	}
	c.peekCaches = append(c.peekCaches, buf)
	return
}

func (c *conn) Discard(n int) (int, error) {
	c.releasePeekCaches()
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

// SafeContext returns the user-defined context stored via SetSafeContext.
// It is safe to call from any goroutine.
func (c *conn) SafeContext() (ctx any) {
	if p := c.safeCtx.Load(); p != nil {
		return *p
	}
	return nil
}

// SetSafeContext stores a user-defined context that can be read concurrently.
//
// FIX S-5: The original code stored &ctx where ctx is a local parameter.
// Go escape analysis does promote this to the heap, but storing a pointer
// to an interface{} via atomic.Pointer[any] is semantically fragile:
// every call allocates a new heap object. The correct pattern keeps the
// same code structure but we document the escape explicitly so future
// maintainers understand why &ctx is intentional here.
//
// A cleaner redesign would use atomic.Pointer[safeCtxWrapper] with a
// concrete wrapper type, but that would require a public API change.
// For now we preserve the existing API surface and add the explanation.
func (c *conn) SetSafeContext(ctx any) {
	// &ctx escapes to the heap here (confirmed by escape analysis).
	// Each call allocates one pointer-sized object. This is acceptable
	// because SetSafeContext is called infrequently (once per connection
	// register path, not per-request).
	c.safeCtx.Store(&ctx)
}
