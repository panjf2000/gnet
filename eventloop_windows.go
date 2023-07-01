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
	"bytes"
	"context"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type eventloop struct {
	ch           chan interface{}   // channel for event-loop
	idx          int                // index of event-loop in event-loops
	eng          *engine            // engine in loop
	cache        bytes.Buffer       // temporary buffer for scattered bytes
	connCount    int32              // number of active connections in event-loop
	connections  map[*conn]struct{} // TCP connection map: fd -> conn
	eventHandler EventHandler       // user eventHandler
}

func (el *eventloop) getLogger() logging.Logger {
	return el.eng.opts.Logger
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
		case *conn:
			err = el.open(v)
		case *tcpConn:
			unpackTCPConn(v)
			err = el.read(v.c)
			resetTCPConn(v)
		case *udpConn:
			err = el.readUDP(v.c)
		case func() error:
			err = v()
		}

		if err == errors.ErrEngineShutdown {
			el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
			break
		} else if err != nil {
			el.getLogger().Debugf("event-loop(%d) got a nonlethal error: %v", el.idx, err)
		}
	}

	return nil
}

func (el *eventloop) open(c *conn) error {
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
		return errors.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer.B)
	c.buffer.Reset()

	return nil
}

func (el *eventloop) readUDP(c *conn) error {
	action := el.eventHandler.OnTraffic(c)
	if action == Shutdown {
		return errors.ErrEngineShutdown
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
		case None:
		case Shutdown:
			if !shutdown {
				shutdown = true
				el.ch <- errors.ErrEngineShutdown
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
	if addr := c.localAddr; addr != nil && strings.HasPrefix(addr.Network(), "udp") {
		action := el.eventHandler.OnClose(c, err)
		if c.rawConn != nil {
			if err := c.rawConn.Close(); err != nil {
				el.getLogger().Errorf("failed to close connection(%s), error:%v", c.remoteAddr.String(), err)
			}
		}
		c.release()
		return el.handleAction(c, action)
	}

	if _, ok := el.connections[c]; !ok {
		return nil // ignore stale wakes.
	}

	delete(el.connections, c)
	el.incConn(-1)
	action := el.eventHandler.OnClose(c, err)
	if err := c.rawConn.Close(); err != nil {
		el.getLogger().Errorf("failed to close connection(%s), error:%v", c.remoteAddr.String(), err)
	}
	c.release()

	return el.handleAction(c, action)
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return errors.ErrEngineShutdown
	default:
		return nil
	}
}
