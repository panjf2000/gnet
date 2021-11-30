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
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/internal/io"
	"github.com/panjf2000/gnet/internal/netpoll"
	gerrors "github.com/panjf2000/gnet/pkg/errors"
	"github.com/panjf2000/gnet/pkg/logging"
)

type eventloop struct {
	ln           *listener       // listener
	idx          int             // loop index in the server loops list
	svr          *server         // server in loop
	poller       *netpoll.Poller // epoll or kqueue
	buffer       []byte          // read packet buffer whose capacity is set by user, default value is 64KB
	connCount    int32           // number of active connections in event-loop
	connections  map[int]*conn   // loop connections fd -> conn
	eventHandler EventHandler    // user eventHandler
}

func (el *eventloop) getLogger() logging.Logger {
	return el.svr.opts.Logger
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

func (el *eventloop) loopRegister(itf interface{}) error {
	c := itf.(*conn)
	if err := el.poller.AddRead(c.pollAttachment); err != nil {
		_ = unix.Close(c.fd)
		c.releaseTCP()
		return err
	}
	el.connections[c.fd] = c
	return el.loopOpen(c)
}

func (el *eventloop) loopOpen(c *conn) error {
	c.opened = true
	el.addConn(1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() {
		if err := el.poller.AddWrite(c.pollAttachment); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(c *conn) error {
	n, err := c.inboundBuffer.CopyFromSocket(c.fd)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return el.loopCloseConn(c, os.NewSyscallError("read", err))
	}
	for packet, _ := c.read(); packet != nil; packet, _ = c.read() {
		out, action := el.eventHandler.React(packet, c)
		if out != nil {
			// Encode data and try to write it back to the peer, this attempt is based on a fact:
			// the peer socket waits for the response data after sending request data to the server,
			// which makes the peer socket writable.
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

		// Check the status of connection every loop since it might be closed
		// during writing data back to the peer due to some kind of system error.
		if !c.opened {
			return nil
		}
	}
	_ = c.inboundBuffer.Rewind()
	return nil
}

func (el *eventloop) loopWrite(c *conn) error {
	el.eventHandler.PreWrite(c)

	iov := c.outboundBuffer.Peek()
	n, err := io.Writev(c.fd, iov)
	c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.loopCloseConn(c, os.NewSyscallError("write", err))
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.pollAttachment)
	}

	return nil
}

func (el *eventloop) loopCloseConn(c *conn, err error) (rerr error) {
	if !c.opened {
		return
	}

	// Send residual data in buffer back to the peer before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		el.eventHandler.PreWrite(c)
		iov := c.outboundBuffer.Peek()
		_, _ = io.Writev(c.fd, iov)
		c.outboundBuffer.Release()
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
			err := el.poller.UrgentTrigger(func(_ interface{}) error { return gerrors.ErrServerShutdown }, nil)
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Tick(), UrgentTrigger:%v", el.idx, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Server, error:%v", el.idx, ctx.Err())
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

	c := newUDPConn(fd, el, el.ln.lnaddr, sa)
	out, action := el.eventHandler.React(el.buffer[:n], c)
	if out != nil {
		_ = c.sendTo(out)
	}
	if action == Shutdown {
		return gerrors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
