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

// +build linux freebsd dragonfly darwin

package gnet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	gerrors "github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/logging"
	"github.com/panjf2000/gnet/internal/netpoll"
	"github.com/panjf2000/gnet/internal/socket"
	"golang.org/x/sys/unix"
)

type eventloop struct {
	internalEventloop

	// Prevents eventloop from false sharing by padding extra memory with the difference
	// between the cache line size "s" and (eventloop mod s) for the most common CPU architectures.
	_ [64 - unsafe.Sizeof(internalEventloop{})%64]byte
}

//nolint:structcheck
type internalEventloop struct {
	ln           *listener       // listener
	idx          int             // loop index in the server loops list
	svr          *server         // server in loop
	poller       *netpoll.Poller // epoll or kqueue
	buffer       []byte          // read packet buffer whose capacity is 64KB
	connCount    int32           // number of active connections in event-loop
	connections  map[int]*conn   // loop connections fd -> conn
	eventHandler EventHandler    // user eventHandler
}

func (el *eventloop) addConn(delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}

func (el *eventloop) loadConn() int32 {
	return atomic.LoadInt32(&el.connCount)
}

func (el *eventloop) closeAllConns() {
	// Close loops and all outstanding connections
	for _, c := range el.connections {
		_ = el.loopCloseConn(c, nil)
	}
}

func (el *eventloop) loopRun(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() {
		el.closeAllConns()
		el.ln.close()
		el.svr.signalShutdown()
	}()

	err := el.poller.Polling(el.handleEvent)
	logging.Infof("Event-loop(%d) is exiting due to error: %v", el.idx, err)
}

func (el *eventloop) loopAccept(fd int) error {
	if fd == el.ln.fd {
		if el.ln.network == "udp" {
			return el.loopReadUDP(fd)
		}

		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return os.NewSyscallError("accept", err)
		}
		if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
			return err
		}

		netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
		c := newTCPConn(nfd, el, sa, netAddr)
		if err = el.poller.AddRead(c.fd); err == nil {
			el.connections[c.fd] = c
			return el.loopOpen(c)
		}
		return err
	}

	return nil
}

func (el *eventloop) loopOpen(c *conn) error {
	c.opened = true
	el.addConn(1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		c.open(out)
	}

	if !c.outboundBuffer.IsEmpty() {
		_ = el.poller.AddWrite(c.fd)
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(c *conn) error {
	n, err := unix.Read(c.fd, el.buffer)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return el.loopCloseConn(c, os.NewSyscallError("read", err))
	}
	c.buffer = el.buffer[:n]

	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			el.eventHandler.PreWrite()
			// Encode data and try to write it back to the client, this attempt is based on a fact:
			// a client socket waits for the response data after sending request data to the server,
			// which makes the client socket writable.
			if err = c.write(out); err != nil {
				return err
			}
		}
		switch action {
		case None:
		case Close:
			return el.loopCloseConn(c, nil)
		case Shutdown:
			return gerrors.ErrServerShutdown
		}

		// Check the status of connection every loop since it might be closed during writing data back to client due to
		// some kind of system error.
		if !c.opened {
			return nil
		}
	}
	_, _ = c.inboundBuffer.Write(c.buffer)

	return nil
}

func (el *eventloop) loopWrite(c *conn) error {
	el.eventHandler.PreWrite()

	head, tail := c.outboundBuffer.LazyReadAll()
	n, err := unix.Write(c.fd, head)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return el.loopCloseConn(c, os.NewSyscallError("write", err))
	}
	c.outboundBuffer.Shift(n)

	if n == len(head) && tail != nil {
		n, err = unix.Write(c.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return el.loopCloseConn(c, os.NewSyscallError("write", err))
		}
		c.outboundBuffer.Shift(n)
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.fd)
	}

	return nil
}

func (el *eventloop) loopCloseConn(c *conn, err error) (rerr error) {
	if !c.opened {
		return nil
	}

	// Send residual data in buffer back to client before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		el.eventHandler.PreWrite()

		head, tail := c.outboundBuffer.LazyReadAll()
		if n, err := unix.Write(c.fd, head); err == nil {
			if n == len(head) && tail != nil {
				_, _ = unix.Write(c.fd, tail)
			}
		}
	}

	if err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd); err0 == nil && err1 == nil {
		delete(el.connections, c.fd)
		el.addConn(-1)

		if el.eventHandler.OnClosed(c, err) == Shutdown {
			return gerrors.ErrServerShutdown
		}
		c.releaseTCP()
	} else {
		if err0 != nil {
			rerr = fmt.Errorf("failed to delete fd=%d from poller in event-loop(%d): %v", c.fd, el.idx, err0)
		}
		if err1 != nil {
			err1 = fmt.Errorf("failed to close fd=%d in event-loop(%d): %v", c.fd, el.idx, os.NewSyscallError("close", err1))
			if rerr != nil {
				rerr = errors.New(rerr.Error() + " & " + err1.Error())
			} else {
				rerr = err1
			}
		}
	}

	return
}

func (el *eventloop) loopWake(c *conn) error {
	if co, ok := el.connections[c.fd]; !ok || co != c {
		return nil // ignore stale wakes.
	}

	out, action := el.eventHandler.React(nil, c)
	if out != nil {
		if err := c.write(out); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopTicker(ctx context.Context) {
	if el == nil {
		return
	}
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
		delay, action = el.eventHandler.Tick()
		switch action {
		case None:
		case Shutdown:
			_ = el.poller.Trigger(func() error { return gerrors.ErrServerShutdown })
			// logging.Debugf("stopping ticker in event-loop(%d) from Tick(), Trigger:%v", el.idx, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			// logging.Debugf("stopping ticker in event-loop(%d) from Server, error:%v", el.idx, ctx.Err())
			return
		case <-timer.C:
		}
	}
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.loopCloseConn(c, nil)
	case Shutdown:
		return gerrors.ErrServerShutdown
	default:
		return nil
	}
}

func (el *eventloop) loopReadUDP(fd int) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil
		}
		return fmt.Errorf("failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err))
	}

	c := newUDPConn(fd, el, sa)
	out, action := el.eventHandler.React(el.buffer[:n], c)
	if out != nil {
		el.eventHandler.PreWrite()
		_ = c.sendTo(out)
	}
	if action == Shutdown {
		return gerrors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
