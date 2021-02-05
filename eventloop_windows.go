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
	"runtime"
	"time"
	"unsafe"

	"github.com/panjf2000/gnet/pool/bytebuffer"

	"github.com/panjf2000/gnet/errors"
)

type eventloop struct {
	internalEventloop

	// Prevents eventloop from false sharing by padding extra memory with the difference
	// between the cache line size "s" and (eventloop mod s) for the most common CPU architectures.
	_ [64 - unsafe.Sizeof(internalEventloop{})%64]byte
}

type internalEventloop struct {
	ch                chan interface{}        // command channel
	idx               int                     // loop index
	svr               *server                 // server in loop
	connCount         int32                   // number of active connections in event-loop
	connections       map[*stdConn]struct{}   // track all the sockets bound to this loop
	eventHandler      EventHandler            // user eventHandler
	calibrateCallback func(*eventloop, int32) // callback func for re-adjusting connCount
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

	for v := range el.ch {
		switch v := v.(type) {
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
		case wakeReq:
			err = el.loopWake(v.c)
		case func() error:
			err = v()
		}

		if err == errors.ErrServerShutdown {
			break
		} else if err != nil {
			el.svr.logger.Infof("Event-loop(%d) is exiting due to the error: %v", el.idx, err)
		}
	}
}

func (el *eventloop) loopAccept(c *stdConn) error {
	el.connections[c] = struct{}{}
	el.calibrateCallback(el, 1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		el.eventHandler.PreWrite()
		_, _ = c.conn.Write(out)
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(c *stdConn) (err error) {
	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			outFrame, _ := c.codec.Encode(c, out)
			el.eventHandler.PreWrite()
			if _, err = c.conn.Write(outFrame); err != nil {
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

	return
}

func (el *eventloop) loopCloseConn(c *stdConn) error {
	return c.conn.SetReadDeadline(time.Now())
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

func (el *eventloop) loopTicker() {
	var (
		delay time.Duration
		open  bool
	)
	for {
		el.ch <- func() (err error) {
			delay, action := el.eventHandler.Tick()
			el.svr.ticktock <- delay
			switch action {
			case Shutdown:
				err = errors.ErrServerShutdown
			}
			return
		}
		if delay, open = <-el.svr.ticktock; open {
			time.Sleep(delay)
		} else {
			break
		}
	}
}

func (el *eventloop) loopError(c *stdConn, err error) (e error) {
	defer func() {
		if err := c.conn.Close(); err != nil {
			el.svr.logger.Warnf("Failed to close connection(%s), error: %v", c.remoteAddr.String(), err)
			if e == nil {
				e = err
			}
		}
		delete(el.connections, c)
		el.calibrateCallback(el, -1)
		c.releaseTCP()
	}()

	switch el.eventHandler.OnClosed(c, err) {
	case Shutdown:
		return errors.ErrServerShutdown
	}

	return
}

func (el *eventloop) loopWake(c *stdConn) error {
	//if co, ok := el.connections[c]; !ok || co != c {
	//	return nil // ignore stale wakes.
	//}
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
	switch action {
	case Shutdown:
		return errors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
