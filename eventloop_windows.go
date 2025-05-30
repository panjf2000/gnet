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
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
	"time"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type eventloop struct {
	ch           chan any           // channel for event-loop
	idx          int                // index of event-loop in event-loops
	eng          *engine            // engine in loop
	connCount    int32              // number of active connections in event-loop
	connections  map[*conn]struct{} // TCP connection map: fd -> conn
	eventHandler EventHandler       // user eventHandler
}

func (el *eventloop) Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error) {
	if el.eng.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if addr == nil {
		return nil, errorx.ErrInvalidNetworkAddress
	}
	return el.enroll(nil, addr, FromContext(ctx))
}

func (el *eventloop) Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error) {
	if el.eng.isShutdown() {
		return nil, errorx.ErrEngineInShutdown
	}
	if c == nil {
		return nil, errorx.ErrInvalidNetConn
	}
	return el.enroll(c, c.RemoteAddr(), FromContext(ctx))
}

func (el *eventloop) Execute(ctx context.Context, runnable Runnable) error {
	if el.eng.isShutdown() {
		return errorx.ErrEngineInShutdown
	}
	if runnable == nil {
		return errorx.ErrNilRunnable
	}
	return goroutine.DefaultWorkerPool.Submit(func() {
		el.ch <- func() error {
			return runnable.Run(ctx)
		}
	})
}

func (el *eventloop) Schedule(context.Context, Runnable, time.Duration) error {
	return errorx.ErrUnsupportedOp
}

func (el *eventloop) Close(c Conn) error {
	return el.close(c.(*conn), nil)
}

func (el *eventloop) getLogger() logging.Logger {
	return el.eng.opts.Logger
}

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

		connOpened := make(chan struct{})
		var gc *conn
		switch addr.Network() {
		case "tcp", "tcp4", "tcp6", "unix":
			gc = newStreamConn(el, c, ctx)
			el.ch <- &openConn{c: gc, cb: func() { close(connOpened) }}
			goroutine.DefaultWorkerPool.Submit(func() {
				var buffer [0x10000]byte
				for {
					n, err := c.Read(buffer[:])
					if err != nil {
						el.ch <- &netErr{gc, err}
						return
					}
					el.ch <- packTCPConn(gc, buffer[:n])
				}
			})
		case "udp", "udp4", "udp6":
			gc = newUDPConn(el, nil, c, c.LocalAddr(), c.RemoteAddr(), ctx)
			el.ch <- &openConn{c: gc, cb: func() { close(connOpened) }}
			goroutine.DefaultWorkerPool.Submit(func() {
				var buffer [0x10000]byte
				for {
					n, err := c.Read(buffer[:])
					if err != nil {
						el.ch <- &netErr{gc, err}
						return
					}
					gc := newUDPConn(el, nil, c, c.LocalAddr(), c.RemoteAddr(), ctx)
					el.ch <- packUDPConn(gc, buffer[:n])
				}
			})
		}

		<-connOpened

		resCh <- RegisteredResult{Conn: gc}
	})
	return
}

func (el *eventloop) incConn(delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}

func (el *eventloop) countConn() int32 {
	return atomic.LoadInt32(&el.connCount)
}

func (el *eventloop) run() (err error) {
	defer func() {
		el.eng.shutdown(err)
		for c := range el.connections {
			_ = el.close(c, nil)
		}
	}()

	if el.eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	for i := range el.ch {
		switch v := i.(type) {
		case error:
			err = v
		case *netErr:
			err = el.close(v.c, v.err)
		case *openConn:
			err = el.open(v)
		case *tcpConn:
			err = el.read(unpackTCPConn(v))
		case *udpConn:
			err = el.readUDP(v.c)
		case func() error:
			err = v()
		}

		if errors.Is(err, errorx.ErrEngineShutdown) {
			el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
			break
		} else if err != nil {
			el.getLogger().Debugf("event-loop(%d) got a nonlethal error: %v", el.idx, err)
		}
	}

	return nil
}

func (el *eventloop) open(oc *openConn) error {
	if oc.cb != nil {
		defer oc.cb()
	}

	c := oc.c
	el.connections[c] = struct{}{}
	el.incConn(1)

	out, action := el.eventHandler.OnOpen(c)
	if out != nil {
		if _, err := c.rawConn.Write(out); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read(c *conn) error {
	if _, ok := el.connections[c]; !ok {
		return nil // ignore stale wakes.
	}
	action := el.eventHandler.OnTraffic(c)
	switch action {
	case None:
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return errorx.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer.B)
	c.buffer.Reset()

	return nil
}

func (el *eventloop) readUDP(c *conn) error {
	action := el.eventHandler.OnTraffic(c)
	if action == Shutdown {
		return errorx.ErrEngineShutdown
	}
	c.release()
	return nil
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
	var shutdown bool
	for {
		delay, action = el.eventHandler.OnTick()
		switch action {
		case None, Close:
		case Shutdown:
			if !shutdown {
				shutdown = true
				el.ch <- errorx.ErrEngineShutdown
				el.getLogger().Debugf("stopping ticker in event-loop(%d) from Tick()", el.idx)
			}
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

func (el *eventloop) wake(c *conn) error {
	if _, ok := el.connections[c]; !ok {
		return nil // ignore stale wakes.
	}
	action := el.eventHandler.OnTraffic(c)
	return el.handleAction(c, action)
}

func (el *eventloop) close(c *conn, err error) error {
	if _, ok := el.connections[c]; c.rawConn == nil || !ok {
		return nil // ignore stale wakes.
	}

	delete(el.connections, c)
	el.incConn(-1)
	action := el.eventHandler.OnClose(c, err)
	err = c.rawConn.Close()
	c.release()
	if err != nil {
		return fmt.Errorf("failed to close connection=%s in event-loop(%d): %v", c.remoteAddr, el.idx, err)
	}

	return el.handleAction(c, action)
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
