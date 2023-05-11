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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/io"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/queue"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/gfd"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

const (
	triggerTypeAsyncWrite = iota
	triggerTypeAsyncWritev
	triggerTypeWake
	triggerTypeClose
	triggerTypeShutdown
	triggerRegister
)

type eventloop struct {
	ln           *listener                  // listener
	idx          int                        // loop index in the engine loops list
	cache        bytes.Buffer               // temporary buffer for scattered bytes
	engine       *engine                    // engine in loop
	poller       *netpoll.Poller            // epoll or kqueue
	buffer       []byte                     // read packet buffer whose capacity is set by user, default value is 64KB
	connCounts   [gfd.ConnIndex1Max]int32   // number of active connections in event-loop
	connNAI1     int                        // connections Next Available Index1
	connNAI2     int                        // connections Next Available Index2
	connSlice    [gfd.ConnIndex1Max][]*conn // connection slice *conn
	connections  map[int]gfd.GFD            // connection map: fd -> GFD
	eventHandler EventHandler               // user eventHandler
}

func (el *eventloop) getLogger() logging.Logger {
	return el.engine.opts.Logger
}

func (el *eventloop) addConn(i1 int, delta int32) {
	atomic.AddInt32(&el.connCounts[i1], delta)
}

func (el *eventloop) loadConn() (ct int32) {
	for i := 0; i < len(el.connCounts); i++ {
		ct += atomic.LoadInt32(&el.connCounts[i])
	}
	return
}

func (el *eventloop) closeAllSockets() {
	// Close loops and all outstanding connections
	for k, cl := range el.connSlice {
		if el.connCounts[k] == 0 {
			continue
		}
		for _, c := range cl {
			if c != nil {
				_ = el.closeConn(c, nil)
			}
		}
	}
}

func (el *eventloop) register(c *conn) error {
	if err := el.poller.AddRead(&c.pollAttachment); err != nil {
		_ = unix.Close(c.fd)
		c.release()
		return err
	}

	el.storeConn(c)
	if c.isDatagram {
		return nil
	}
	return el.open(c)
}

// storeConn store conn
// find the next available location
// 1. current space available location
// 2. allocated other space is available
// 3. unallocated space (reapply when using it)
// 4. if no usable space is found, return directly.
func (el *eventloop) storeConn(c *conn) {
	if el.connNAI1 >= gfd.ConnIndex1Max {
		return
	}

	if el.connSlice[el.connNAI1] == nil {
		el.connSlice[el.connNAI1] = make([]*conn, gfd.ConnIndex2Max)
	}
	el.connSlice[el.connNAI1][el.connNAI2] = c

	c.gfd = gfd.NewGFD(c.fd, el.idx, el.connNAI1, el.connNAI2)
	el.connections[c.fd] = c.gfd
	el.addConn(el.connNAI1, 1)

	for i2 := el.connNAI2; i2 < gfd.ConnIndex2Max; i2++ {
		if el.connSlice[el.connNAI1][i2] == nil {
			el.connNAI2 = i2
			return
		}
	}

	for i1 := el.connNAI1; i1 < gfd.ConnIndex1Max; i1++ {
		if el.connSlice[i1] == nil || el.connCounts[i1] >= gfd.ConnIndex2Max {
			continue
		}
		for i2 := 0; i2 < gfd.ConnIndex2Max; i2++ {
			if el.connSlice[i1][i2] == nil {
				el.connNAI1, el.connNAI2 = i1, i2
				return
			}
		}
	}

	for i1 := 0; i1 < gfd.ConnIndex1Max; i1++ {
		if el.connSlice[i1] == nil {
			el.connNAI1, el.connNAI2 = i1, 0
			return
		}
	}

	el.connNAI1 = gfd.ConnIndex1Max
}

func (el *eventloop) removeConn(c *conn) {
	delete(el.connections, c.fd)
	el.addConn(c.gfd.ConnIndex1(), -1)
	if el.connCounts[c.gfd.ConnIndex1()] == 0 {
		el.connSlice[c.gfd.ConnIndex1()] = nil
	} else {
		el.connSlice[c.gfd.ConnIndex1()][c.gfd.ConnIndex2()] = nil
	}

	if el.connNAI1 > c.gfd.ConnIndex1() || el.connNAI2 > c.gfd.ConnIndex2() {
		el.connNAI1, el.connNAI2 = c.gfd.ConnIndex1(), c.gfd.ConnIndex2()
	}
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
		return el.closeConn(c, os.NewSyscallError("read", err))
	}

	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	switch action {
	case None:
	case Close:
		return el.closeConn(c, nil)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	c.buffer = c.buffer[:0]
	return nil
}

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
		return el.closeConn(c, os.NewSyscallError("write", err))
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(&c.pollAttachment)
	}

	return nil
}

func (el *eventloop) closeConn(c *conn, err error) (rerr error) {
	if addr := c.localAddr; addr != nil && strings.HasPrefix(c.localAddr.Network(), "udp") {
		rerr = el.poller.Delete(c.fd)
		if c.fd != el.ln.fd {
			rerr = unix.Close(c.fd)
			el.removeConn(c)
		}
		if el.eventHandler.OnClose(c, err) == Shutdown {
			return gerrors.ErrEngineShutdown
		}
		c.release()
		return
	}

	if !c.opened {
		return
	}

	// Send residual data in buffer back to the peer before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		for !c.outboundBuffer.IsEmpty() {
			iov := c.outboundBuffer.Peek(0)
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			if n, e := io.Writev(c.fd, iov); e != nil {
				el.getLogger().Warnf("closeConn: error occurs when sending data back to peer, %v", e)
				break
			} else {
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

	el.removeConn(c)
	if el.eventHandler.OnClose(c, err) == Shutdown {
		rerr = gerrors.ErrEngineShutdown
	}
	c.release()

	return
}

func (el *eventloop) wake(c *conn) error {
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
			err := el.poller.UrgentTrigger(triggerTypeShutdown, gfd.GFD{}, nil)
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

func (el *eventloop) taskRun(task *queue.Task) (err error) {
	switch task.TaskType {
	case triggerTypeShutdown:
		return gerrors.ErrEngineShutdown
	case triggerRegister:
		return el.register(task.Arg.(*conn))
	}

	if el.connSlice[task.GFD.ConnIndex1()] == nil {
		return
	}
	c := el.connSlice[task.GFD.ConnIndex1()][task.GFD.ConnIndex2()]
	if c == nil || c.gfd != task.GFD {
		return
	}
	switch task.TaskType {
	case triggerTypeAsyncWrite:
		return c.asyncWrite(task.Arg.(*asyncWriteHook))
	case triggerTypeAsyncWritev:
		return c.asyncWritev(task.Arg.(*asyncWritevHook))
	case triggerTypeClose:
		err = el.closeConn(c, nil)
		if task.Arg != nil {
			if callback, ok := task.Arg.(AsyncCallback); ok && callback != nil {
				_ = callback(c, err)
			}
		}
		return
	case triggerTypeWake:
		err = el.wake(c)
		if task.Arg != nil {
			if callback, ok := task.Arg.(AsyncCallback); ok && callback != nil {
				_ = callback(c, err)
			}
		}
		return
	default:
		return
	}
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.closeConn(c, nil)
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
		gFd := el.connections[fd]
		c = el.connSlice[gFd.ConnIndex1()][gFd.ConnIndex2()]
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
