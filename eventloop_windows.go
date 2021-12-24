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

package gnet

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/pkg/errors"
	"github.com/panjf2000/gnet/pkg/logging"
	bbPool "github.com/panjf2000/gnet/pkg/pool/bytebuffer"
)

type eventloop struct {
	ch           chan interface{}      // command channel
	idx          int                   // loop index
	svr          *server               // server in loop
	connCount    int32                 // number of active connections in event-loop
	connections  map[*stdConn]struct{} // track all the sockets bound to this loop
	eventHandler EventHandler          // user eventHandler
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

func (el *eventloop) run(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	var err error
	defer func() {
		el.svr.signalShutdownWithErr(err)
		el.svr.loopWG.Done()
		el.egress()
		el.svr.loopWG.Done()
	}()

	for i := range el.ch {
		switch v := i.(type) {
		case error:
			err = v
		case *stdConn:
			err = el.accept(v)
		case *tcpConn:
			if v.c.conn == nil {
				err = errors.ErrConnectionClosed
				break
			}
			v.c.buffer = v.bb
			err = el.read(v.c)
		case *udpConn:
			err = el.readUDP(v.c)
		case *stderr:
			err = el.error(v.c, v.err)
		case *signalTask:
			err = v.run(v.c)
			signalTaskPool.Put(i)
		case *dataTask:
			_, err = v.run(v.buf)
			dataTaskPool.Put(i)
		}

		if err == errors.ErrServerShutdown {
			el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
			break
		} else if err != nil {
			el.getLogger().Debugf("event-loop(%d) got a nonlethal error: %v", el.idx, err)
		}
	}
}

func (el *eventloop) accept(c *stdConn) error {
	el.connections[c] = struct{}{}
	el.addConn(1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		el.eventHandler.PreWrite(c)
		_, _ = c.conn.Write(out)
		el.eventHandler.AfterWrite(c, out)
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read(c *stdConn) error {
	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			if _, err := c.write(out); err != nil {
				return el.error(c, err)
			}
		}
		switch action {
		case None:
		case Close:
			return el.closeConn(c)
		case Shutdown:
			return errors.ErrServerShutdown
		}
	}
	_, _ = c.inboundBuffer.Write(c.buffer.B)
	bbPool.Put(c.buffer)
	c.buffer = nil

	return nil
}

func (el *eventloop) closeConn(c *stdConn) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(time.Now())
	}
	return nil
}

func (el *eventloop) egress() {
	var closed bool
	for v := range el.ch {
		switch v := v.(type) {
		case error:
			if v == errCloseAllConns {
				closed = true
				for c := range el.connections {
					_ = el.closeConn(c)
				}
			}
		case *stderr:
			_ = el.error(v.c, v.err)
		}
		if closed && len(el.connections) == 0 {
			break
		}
	}
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
		delay, action = el.eventHandler.Tick()
		if action == Shutdown {
			el.ch <- errors.ErrServerShutdown
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Tick()", el.idx)
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

func (el *eventloop) error(c *stdConn, err error) (e error) {
	defer func() {
		if _, ok := el.connections[c]; !ok {
			return // ignore stale wakes.
		}

		if err = c.conn.Close(); err != nil {
			el.getLogger().Errorf("failed to close connection(%s), error: %v", c.remoteAddr.String(), err)
			if e == nil {
				e = err
			}
		}
		delete(el.connections, c)
		el.addConn(-1)

		c.releaseTCP()
	}()

	if el.eventHandler.OnClosed(c, err) == Shutdown {
		return errors.ErrServerShutdown
	}

	return
}

func (el *eventloop) wake(c *stdConn) error {
	if _, ok := el.connections[c]; !ok {
		return nil // ignore stale wakes.
	}

	out, action := el.eventHandler.React(nil, c)
	if out != nil {
		if _, err := c.write(out); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) handleAction(c *stdConn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.closeConn(c)
	case Shutdown:
		return errors.ErrServerShutdown
	default:
		return nil
	}
}

func (el *eventloop) readUDP(c *stdConn) error {
	out, action := el.eventHandler.React(c.buffer.Bytes(), c)
	if out != nil {
		_ = c.SendTo(out)
	}
	if action == Shutdown {
		return errors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
