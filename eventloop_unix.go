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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	errorx "github.com/nelthaarion/gnet/v2/pkg/errors"
	gio "github.com/nelthaarion/gnet/v2/pkg/io"
	"github.com/nelthaarion/gnet/v2/pkg/logging"
	"github.com/nelthaarion/gnet/v2/pkg/netpoll"
	"github.com/nelthaarion/gnet/v2/pkg/pool/goroutine"
	"github.com/nelthaarion/gnet/v2/pkg/queue"
	"github.com/nelthaarion/gnet/v2/pkg/socket"
)

type eventloop struct {
	listeners    map[int]*listener // listeners
	idx          int               // loop index in the engine loops list
	engine       *engine           // engine in loop
	poller       *netpoll.Poller   // epoll or kqueue
	buffer       []byte            // read packet buffer whose capacity is set by user, default value is 64KB
	connections  connMatrix        // loop connections storage
	eventHandler EventHandler      // user eventHandler
	// gfdSeq feeds the gfd of UDP connections, which this loop creates once
	// per datagram; a per-loop counter keeps the global one off that path.
	gfdSeq uint32
}

func (el *eventloop) Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if addr == nil {
		return nil, errorx.ErrInvalidNetworkAddress
	}
	return el.enroll(nil, addr, FromContext(ctx))
}

func (el *eventloop) Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if c == nil {
		return nil, errorx.ErrInvalidNetConn
	}
	return el.enroll(c, c.RemoteAddr(), FromContext(ctx))
}

func (el *eventloop) Execute(ctx context.Context, runnable Runnable) error {
	if el.engine.isShutdown() {
		return errorx.ErrEngineInShutdown
	}
	if runnable == nil {
		return errorx.ErrNilRunnable
	}
	return el.poller.Trigger(queue.LowPriority, func(any) error {
		return runnable.Run(ctx)
	}, nil)
}

func (el *eventloop) Schedule(context.Context, Runnable, time.Duration) error {
	return errorx.ErrUnsupportedOp
}

func (el *eventloop) Close(c Conn) error {
	return el.close(c.(*conn), nil)
}

func (el *eventloop) getLogger() logging.Logger {
	return el.engine.opts.Logger
}

// pollYield reports whether the poller of this event-loop may yield to the Go
// scheduler before it blocks, see Options.NoPollYield.
func (el *eventloop) pollYield() bool {
	return !el.engine.opts.NoPollYield
}

// setupAdaptivePollYield installs the yield policy of a sub-reactor, a loop that serves
// the connections the main reactor hands to it (orbit). Unless NoPollYield is set the
// policy is adaptive: the loop yields while other goroutines are handing work to it and
// skips the yield, and the extra system call that comes with it, while it serves its
// traffic inline. That only happens in a process that has more Ps than event-loops: a
// goroutine that a callback makes runnable can then always go to a P that no loop is
// using, whereas with a P per loop (what Multicore gives) the yield is how such a
// goroutine gets to run, and the benchmarks gave no reason to take it away there. A loop
// that is locked to its thread keeps the unconditional yield too: it is what hands the P
// over to the next locked loop in a process that has fewer Ps than locked loops.
// AlwaysPollYield asks for the unconditional yield.
//
// The pollers that accept connections themselves (the main reactor, rotate, and the loops
// of a ReusePort server, run) are not adaptive: they are woken by the kernel and not
// through the task queues, so nothing would ever tell them that work is flowing, and on
// connection-churn workloads a main reactor that skipped the yield cost a fifth to a
// quarter of the throughput in the benchmarks.
func (el *eventloop) setupAdaptivePollYield() {
	opts := el.engine.opts
	el.poller.SetYield(el.pollYield())
	el.poller.SetAdaptiveYield(!opts.LockOSThread && !opts.AlwaysPollYield &&
		runtime.GOMAXPROCS(0) > determineEventLoops(opts))
}

func (el *eventloop) countConn() int32 {
	return el.connections.loadCount()
}

func (el *eventloop) closeConns() {
	// Close loops and all outstanding connections
	el.connections.iterate(func(c *conn) bool {
		_ = el.close(c, nil)
		return true
	})
}

type connWithCallback struct {
	c  *conn
	cb func()
}

// enroll dials or wraps an existing net.Conn and registers it with this event-loop.
//
// FIX S-6: The previous code submitted the dial+register work to goroutine.DefaultWorkerPool
// (which is non-blocking). If the pool was full it returned an error but resCh was never
// closed or written to, so Engine.Register() callers would block forever waiting on the channel.
//
// Fix: on Submit failure, immediately send the error on resCh and close it so callers
// always receive exactly one value and the channel is GC-able.
func (el *eventloop) enroll(c net.Conn, addr net.Addr, ctx any) (resCh chan RegisteredResult, err error) {
	resCh = make(chan RegisteredResult, 1)
	err = goroutine.DefaultWorkerPool.Submit(func() {
		defer close(resCh)

		var err error
		if c == nil {
			if c, err = net.Dial(addr.Network(), addr.String()); err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
		}
		defer c.Close() //nolint:errcheck

		sc, ok := c.(syscall.Conn)
		if !ok {
			resCh <- RegisteredResult{
				Err: fmt.Errorf("failed to assert syscall.Conn from net.Conn: %s", addr.String()),
			}
			return
		}
		rc, err := sc.SyscallConn()
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}

		var dupFD int
		err1 := rc.Control(func(fd uintptr) {
			dupFD, err = socket.Dup(int(fd))
		})
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}
		if err1 != nil {
			resCh <- RegisteredResult{Err: err1}
			return
		}

		var (
			sockAddr unix.Sockaddr
			gc       *conn
		)
		switch c.(type) {
		case *net.UnixConn:
			sockAddr, _, _, err = socket.GetUnixSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			ua := c.LocalAddr().(*net.UnixAddr)
			ua.Name = c.RemoteAddr().String() + "." + strconv.Itoa(dupFD)
			gc = newStreamConn("unix", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())
		case *net.TCPConn:
			sockAddr, _, _, _, err = socket.GetTCPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newStreamConn("tcp", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())
		case *net.UDPConn:
			sockAddr, _, _, _, err = socket.GetUDPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newUDPConn(dupFD, el, c.LocalAddr(), sockAddr, true)
		default:
			resCh <- RegisteredResult{Err: fmt.Errorf("unknown type of conn: %T", c)}
			return
		}

		gc.SetContext(ctx)
		gc.SetSafeContext(ctx)

		connOpened := make(chan struct{})
		ccb := &connWithCallback{c: gc, cb: func() {
			close(connOpened)
		}}
		if err := el.poller.Trigger(queue.LowPriority, el.register, ccb); err != nil {
			gc.Close() //nolint:errcheck
			resCh <- RegisteredResult{Err: err}
			return
		}
		<-connOpened

		resCh <- RegisteredResult{Conn: gc}
	})

	// FIX S-6: If the goroutine pool is full (non-blocking mode returns error),
	// enroll() previously returned a non-nil err with an open, empty resCh.
	// Any caller doing <-resCh would deadlock. Send the error and close the
	// channel so callers always get exactly one result.
	if err != nil {
		resCh <- RegisteredResult{Err: fmt.Errorf("goroutine pool is full, failed to enroll connection: %w", err)}
		close(resCh)
	}
	return
}

func (el *eventloop) register(a any) error {
	c, ok := a.(*conn)
	if !ok {
		ccb := a.(*connWithCallback)
		c = ccb.c
		defer ccb.cb()
	}
	return el.register0(c)
}

func (el *eventloop) register0(c *conn) error {
	addEvents := el.poller.AddRead
	if el.engine.opts.EdgeTriggeredIO {
		addEvents = el.poller.AddReadWrite
	}
	if err := addEvents(&c.pollAttachment, el.engine.opts.EdgeTriggeredIO); err != nil {
		_ = unix.Close(c.fd)
		c.release()
		return err
	}
	if err := el.connections.addConn(c, el.idx); err != nil {
		// The matrix has no free slot left.  The poller registration above already
		// succeeded, so deregister the fd before failing: otherwise the event-loop
		// would keep receiving events for a connection it does not track.
		_ = el.poller.Delete(c.fd)
		_ = unix.Close(c.fd)
		c.release()
		return err
	}
	if c.isDatagram && c.remote != nil {
		return nil
	}
	return el.open(c)
}

func (el *eventloop) open(c *conn) error {
	c.opened = true

	out, action := el.eventHandler.OnOpen(c)
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() && !el.engine.opts.EdgeTriggeredIO {
		if err := el.poller.ModReadWrite(&c.pollAttachment, false); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read0(a any) error {
	return el.read(a.(*conn))
}

func (el *eventloop) read(c *conn) error {
	if !c.opened {
		return nil
	}

	var recv int
	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk
loop:
	n, err := gio.ReadSocketNB(c.fd, el.buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = io.EOF
		}
		return el.close(c, os.NewSyscallError("read", err))
	}
	recv += n

	c.buffer = el.buffer[:n]
	coalesce := !isET && !el.engine.opts.NoWriteCoalescing
	if coalesce {
		c.openCork() // open the write-coalescing window, see corkState
	}
	action := el.eventHandler.OnTraffic(c)
	flush := c.closeCork(coalesce)
	switch action {
	case None:
		if flush && c.opened {
			if err = el.flushCoalesced(c); err != nil || !c.opened {
				return err
			}
		}
	case Close:
		return el.close(c, nil) // close sends whatever the callback left in the outbound buffer
	case Shutdown:
		return errorx.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	c.buffer = c.buffer[:0]

	if c.isEOF || (isET && recv < chunk) {
		goto loop
	}

	// To prevent infinite reading in ET mode and starving other events,
	// we need to set up threshold for the maximum read bytes per connection
	// on each event-loop. If the threshold is reached and there are still
	// unread data in the socket buffer, we must issue another read event manually.
	if isET && n == len(el.buffer) {
		return el.poller.Trigger(queue.LowPriority, el.read0, c)
	}

	return nil
}

func (el *eventloop) write0(a any) error {
	return el.write(a.(*conn))
}

// The default value of UIO_MAXIOV/IOV_MAX is 1024 on Linux and most BSD-like OSs.
const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	if c.outboundBuffer.IsEmpty() {
		return nil
	}

	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk
	var (
		n    int
		sent int
		err  error
	)
loop:
	iov, _ := c.outboundBuffer.Peek(-1)
	// Fast path: Peek always reports the ring-buffer as two slices (head, tail)
	// even when the data is a single contiguous run, so a plain write(2) is the
	// right call whenever the tail is empty. The old code took the writev branch
	// unconditionally in that case, paying an extra iovec and a heavier syscall
	// per flush on the overwhelmingly common single-segment buffer.
	if len(iov) == 2 && len(iov[1]) == 0 {
		n, err = gio.WriteSocketNB(c.fd, iov[0])
	} else if len(iov) > 1 {
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err = gio.Writev(c.fd, iov)
	} else {
		n, err = gio.WriteSocketNB(c.fd, iov[0])
	}
	_, _ = c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.close(c, os.NewSyscallError("write", err))
	}
	sent += n

	if isET && !c.outboundBuffer.IsEmpty() && sent < chunk {
		goto loop
	}

	// All data have been sent, it's no need to monitor the writable events for LT mode,
	// remove the writable event from poller to help the future event-loops if necessary.
	if !isET && c.outboundBuffer.IsEmpty() {
		return el.poller.ModRead(&c.pollAttachment, false)
	}

	// To prevent infinite writing in ET mode and starving other events,
	// we need to set up threshold for the maximum write bytes per connection
	// on each event-loop. If the threshold is reached and there are still
	// pending data to write, we must issue another write event manually.
	if isET && !c.outboundBuffer.IsEmpty() {
		return el.poller.Trigger(queue.HighPriority, el.write0, c)
	}

	return nil
}

// flushCoalesced sends the data that conn.write and conn.writev buffered while an
// OnTraffic callback was running (see corkState). It runs once, right after the
// callback returns, and replaces what would have been one write(2) per Write call
// with as few as one.
//
// Level-triggered mode only: the descriptor is not registered for writable events at
// this point, so whatever the kernel does not take now is handed to the poller by
// switching the registration to read+write, exactly as an immediate write that
// hit a full socket buffer does.
func (el *eventloop) flushCoalesced(c *conn) error {
	for !c.outboundBuffer.IsEmpty() {
		var (
			n   int
			err error
		)
		// Common case: everything sits in the ring part of the buffer, send the
		// first contiguous run with the single-buffer socket syscall and no allocation.
		if head, _, _ := c.outboundBuffer.PeekStatic(); len(head) > 0 {
			n, err = gio.WriteSocketNB(c.fd, head)
			if n > 0 {
				_, _ = c.outboundBuffer.Discard(n)
			}
			if err == nil && n < len(head) {
				break // the socket buffer is full, wait for EPOLLOUT
			}
		} else { // only the list part holds data: gather it with writev
			iov, _ := c.outboundBuffer.Peek(-1)
			if len(iov) == 0 {
				break
			}
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			n, err = gio.Writev(c.fd, iov)
			if n > 0 {
				_, _ = c.outboundBuffer.Discard(n)
			}
			if err == nil && n == 0 {
				break // no progress, do not spin
			}
		}
		if err != nil {
			if err == unix.EAGAIN {
				break
			}
			return el.close(c, os.NewSyscallError("write", err))
		}
	}
	if !c.outboundBuffer.IsEmpty() {
		return el.poller.ModReadWrite(&c.pollAttachment, false)
	}
	return nil
}

func (el *eventloop) close(c *conn, err error) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // ignore stale connections
	}

	el.connections.delConn(c)
	action := el.eventHandler.OnClose(c, err)

	// Send residual data in buffer back to the remote before actually closing the connection.
	// FIX P-6: The previous code silently discarded write errors here (only `break` on error,
	// no logging). Log the error so operators know when data was lost during teardown.
	for !c.outboundBuffer.IsEmpty() {
		iov, _ := c.outboundBuffer.Peek(0)
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, werr := gio.Writev(c.fd, iov)
		if werr != nil {
			el.getLogger().Warnf("failed to flush outbound buffer on close for fd=%d: %v",
				c.fd, os.NewSyscallError("writev", werr))
			break
		}
		_, _ = c.outboundBuffer.Discard(n)
	}

	c.release()

	var errStr strings.Builder
	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 != nil {
		err0 = fmt.Errorf("failed to delete fd=%d from poller in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("delete", err0))
		errStr.WriteString(err0.Error())
		errStr.WriteString(" | ")
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("close", err1))
		errStr.WriteString(err1.Error())
	}
	if errStr.Len() > 0 {
		return errors.New(strings.TrimSuffix(errStr.String(), " | "))
	}

	return el.handleAction(c, action)
}

func (el *eventloop) wake(c *conn) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // ignore stale connections
	}

	coalesce := !el.engine.opts.EdgeTriggeredIO && !el.engine.opts.NoWriteCoalescing
	if coalesce {
		c.openCork() // open the write-coalescing window, see corkState
	}
	action := el.eventHandler.OnTraffic(c)
	flush := c.closeCork(coalesce)
	if flush && action == None && c.opened {
		if err := el.flushCoalesced(c); err != nil || !c.opened {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) ticker(ctx context.Context) {
	var (
		action Action
		delay  time.Duration
		timer  *time.Timer
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		delay, action = el.eventHandler.OnTick()
		switch action {
		case None:
			// no-op: continue ticking
		case Close:
			// FIX P-5: The original code lumped Close with None (case None, Close:)
			// causing Close returned from OnTick to be silently ignored.
			// Close has no meaningful target in a ticker (there is no current
			// connection), so we log a warning to alert the developer that their
			// OnTick implementation is returning an unexpected action, then
			// continue as if None was returned.
			el.getLogger().Warnf("OnTick() returned Action.Close which has no effect in a ticker; " +
				"return Action.None to continue ticking or Action.Shutdown to stop the engine")
		case Shutdown:
			// It seems reasonable to mark this as low-priority, waiting for some tasks like asynchronous writes
			// to finish up before shutting down the service.
			err := el.poller.Trigger(queue.LowPriority, func(_ any) error { return errorx.ErrEngineShutdown }, nil)
			el.getLogger().Debugf("failed to enqueue shutdown signal of high-priority for event-loop(%d): %v", el.idx, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Engine, error:%v", el.idx, ctx.Err())
			return
		case <-timer.C:
		}
	}
}

func (el *eventloop) readUDP(fd int, _ netpoll.IOEvent, _ netpoll.IOFlags) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return fmt.Errorf("failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err))
	}
	var c *conn
	if ln, ok := el.listeners[fd]; ok {
		c = newUDPConn(fd, el, ln.addr, sa, false)
	} else {
		c = el.connections.getConn(fd)
	}
	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	if c.remote != nil {
		c.release()
	}
	if action == Shutdown {
		return errorx.ErrEngineShutdown
	}
	return nil
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return errorx.ErrEngineShutdown
	default:
		return nil
	}
}
