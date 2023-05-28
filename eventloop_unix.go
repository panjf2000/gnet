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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/io"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type eventloop struct {
	ln           *listener       // listener
	idx          int             // loop index in the engine loops list
	cache        bytes.Buffer    // temporary buffer for scattered bytes
	engine       *engine         // engine in loop
	poller       *netpoll.Poller // epoll or kqueue
	buffer       []byte          // read packet buffer whose capacity is set by user, default value is 64KB
	connections  connMatrix      // loop connections storage
	eventHandler EventHandler    // user eventHandler
}

func (el *eventloop) getLogger() logging.Logger {
	return el.engine.opts.Logger
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

func (el *eventloop) register(itf interface{}) error {
	c := itf.(*conn)
	if err := el.poller.AddRead(&c.pollAttachment); err != nil {
		_ = unix.Close(c.fd)
		c.release()
		return err
	}

	el.connections.addConn(c, el.idx)

	if c.isDatagram {
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

	if !c.outboundBuffer.IsEmpty() {
		if err := el.poller.AddWrite(&c.pollAttachment); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read(c *conn) error {
	n, err := unix.Read(c.fd, el.buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = unix.ECONNRESET
		}
		return el.close(c, os.NewSyscallError("read", err))
	}

	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	switch action {
	case None:
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	c.buffer = c.buffer[:0]
	return nil
}

// The default value of UIO_MAXIOV/IOV_MAX is 1024 on Linux and most BSD-like OSs.
const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	iov := c.outboundBuffer.Peek(-1)
	var (
		n   int
		err error
	)
	if len(iov) > 1 {
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err = io.Writev(c.fd, iov)
	} else {
		n, err = unix.Write(c.fd, iov[0])
	}
	_, _ = c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.close(c, os.NewSyscallError("write", err))
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(&c.pollAttachment)
	}

	return nil
}

func (el *eventloop) close(c *conn, err error) (rerr error) {
	if addr := c.localAddr; addr != nil && strings.HasPrefix(c.localAddr.Network(), "udp") {
		rerr = el.poller.Delete(c.fd)
		if c.fd != el.ln.fd {
			rerr = unix.Close(c.fd)
			el.connections.delConn(c)
		}
		if el.eventHandler.OnClose(c, err) == Shutdown {
			return gerrors.ErrEngineShutdown
		}
		c.release()
		return
	}

	if !c.opened || el.connections.getConn(c.fd) == nil {
		return // ignore stale connections
	}

	// Send residual data in buffer back to the peer before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		for !c.outboundBuffer.IsEmpty() {
			iov := c.outboundBuffer.Peek(0)
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			if n, e := io.Writev(c.fd, iov); e != nil {
				el.getLogger().Warnf("close: error occurs when sending data back to peer, %v", e)
				break
			} else { //nolint:revive
				_, _ = c.outboundBuffer.Discard(n)
			}
		}
	}

	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
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

	el.connections.delConn(c)
	if el.eventHandler.OnClose(c, err) == Shutdown {
		rerr = gerrors.ErrEngineShutdown
	}
	c.release()

	return
}

func (el *eventloop) wake(c *conn) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // ignore stale connections
	}

	action := el.eventHandler.OnTraffic(c)

	return el.handleAction(c, action)
}

func (el *eventloop) ticker(ctx context.Context) {
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
		delay, action = el.eventHandler.OnTick()
		switch action {
		case None:
		case Shutdown:
			err := el.poller.UrgentTrigger(func(_ interface{}) error { return gerrors.ErrEngineShutdown }, nil)
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from OnTick(), UrgentTrigger:%v", el.idx, err)
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

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	default:
		return nil
	}
}

func (el *eventloop) readUDP(fd int, _ netpoll.IOEvent) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil
		}
		return fmt.Errorf("failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err))
	}
	var c *conn
	if fd == el.ln.fd {
		c = newUDPConn(fd, el, el.ln.addr, sa, false)
	} else {
		c = el.connections.getConn(fd)
	}
	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	if c.peer != nil {
		c.release()
	}
	if action == Shutdown {
		return gerrors.ErrEngineShutdown
	}
	return nil
}

/*
func (el *eventloop) execCmd(itf interface{}) (err error) {
	cmd := itf.(*asyncCmd)
	c := el.connections.getConnByGFD(cmd.fd)
	if c == nil || c.gfd != cmd.fd {
		return gerrors.ErrInvalidConn
	}

	defer func() {
		if cmd.cb != nil {
			_ = cmd.cb(c, err)
		}
	}()

	switch cmd.typ {
	case asyncCmdClose:
		return el.close(c, nil)
	case asyncCmdWake:
		return el.wake(c)
	case asyncCmdWrite:
		_, err = c.Write(cmd.arg.([]byte))
	case asyncCmdWritev:
		_, err = c.Writev(cmd.arg.([][]byte))
	default:
		return gerrors.ErrUnsupportedOp
	}
	return
}
*/
