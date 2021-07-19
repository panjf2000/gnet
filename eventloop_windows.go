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

package gnet

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/logging"
	"github.com/panjf2000/gnet/pool/bytebuffer"
)

type eventloop struct {
	internalEventloop

	// Prevents eventloop from false sharing by padding extra memory with the difference
	// between the cache line size "s" and (eventloop mod s) for the most common CPU architectures.
	_ [64 - unsafe.Sizeof(internalEventloop{})%64]byte
}

//nolint:structcheck
type internalEventloop struct {
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

func (el *eventloop) loopRun(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	var err error
	defer func() {
		el.svr.signalShutdownWithErr(err)
		el.svr.loopWG.Done()
		el.loopEgress()
		el.svr.loopWG.Done()
	}()

	for i := range el.ch {
		switch v := i.(type) {
		case error:
			err = v
		case *stdConn:
			err = el.loopAccept(v)
		case *tcpConn:
			v.c.buffer = v.bb
			err = el.loopRead(v.c)
		case *udpConn:
			err = el.loopReadUDP(v.c)
		case *stderr:
			err = el.loopError(v.c, v.err)
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
			el.getLogger().Errorf("event-loop(%d) is exiting due to the error: %v", el.idx, err)
		}
	}
}

func (el *eventloop) loopAccept(c *stdConn) error {
	el.connections[c] = struct{}{}
	el.addConn(1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		el.eventHandler.PreWrite()
		_, _ = c.conn.Write(out)
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(c *stdConn) error {
	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			outFrame, _ := c.codec.Encode(c, out)
			el.eventHandler.PreWrite()
			if _, err := c.conn.Write(outFrame); err != nil {
				return el.loopError(c, err)
			}
		}
		switch action {
		case None:
		case Close:
			return el.loopCloseConn(c)
		case Shutdown:
			return errors.ErrServerShutdown
		}
	}
	_, _ = c.inboundBuffer.Write(c.buffer.Bytes())
	bytebuffer.Put(c.buffer)
	c.buffer = nil

	return nil
}

func (el *eventloop) loopCloseConn(c *stdConn) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(time.Now())
	}
	return nil
}

func (el *eventloop) loopEgress() {
	var closed bool
	for v := range el.ch {
		switch v := v.(type) {
		case error:
			if v == errCloseAllConns {
				closed = true
				for c := range el.connections {
					_ = el.loopCloseConn(c)
				}
			}
		case *stderr:
			_ = el.loopError(v.c, v.err)
		}
		if closed && len(el.connections) == 0 {
			break
		}
	}
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

func (el *eventloop) loopError(c *stdConn, err error) (e error) {
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

func (el *eventloop) loopWake(c *stdConn) error {
	if _, ok := el.connections[c]; !ok {
		return nil // ignore stale wakes.
	}

	out, action := el.eventHandler.React(nil, c)
	if out != nil {
		if frame, err := c.codec.Encode(c, out); err != nil {
			return err
		} else if _, err = c.conn.Write(frame); err != nil {
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
		return el.loopCloseConn(c)
	case Shutdown:
		return errors.ErrServerShutdown
	default:
		return nil
	}
}

func (el *eventloop) loopReadUDP(c *stdConn) error {
	out, action := el.eventHandler.React(c.buffer.Bytes(), c)
	if out != nil {
		el.eventHandler.PreWrite()
		_, _ = el.svr.ln.pconn.WriteTo(out, c.remoteAddr)
	}
	if action == Shutdown {
		return errors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
