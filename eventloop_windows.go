// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"net"
	"time"

	"github.com/panjf2000/gnet/pool/bytebuffer"
)

type eventloop struct {
	ch                chan interface{}        // command channel
	idx               int                     // loop index
	svr               *server                 // server in loop
	codec             ICodec                  // codec for TCP
	connCount         int32                   // number of active connections in event-loop
	connections       map[*stdConn]struct{}   // track all the sockets bound to this loop
	eventHandler      EventHandler            // user eventHandler
	calibrateCallback func(*eventloop, int32) // callback func for re-adjusting connCount
}

func (el *eventloop) loopRun() {
	var err error
	defer func() {
		if el.idx == 0 && el.svr.opts.Ticker {
			close(el.svr.ticktock)
		}
		el.svr.signalShutdown(err)
		el.svr.loopWG.Done()
		el.loopEgress()
		el.svr.loopWG.Done()
	}()
	if el.idx == 0 && el.svr.opts.Ticker {
		go el.loopTicker()
	}
	for v := range el.ch {
		switch v := v.(type) {
		case error:
			err = v
		case *stdConn:
			err = el.loopAccept(v)
		case *tcpIn:
			err = el.loopRead(v)
		case *udpIn:
			err = el.loopReadUDP(v.c)
		case *stderr:
			err = el.loopError(v.c, v.err)
		case wakeReq:
			err = el.loopWake(v.c)
		case func() error:
			err = v()
		}
		if err != nil {
			el.svr.logger.Printf("event-loop:%d exits with error:%v\n", el.idx, err)
			break
		}
	}
}

func (el *eventloop) loopAccept(c *stdConn) error {
	el.connections[c] = struct{}{}
	c.localAddr = el.svr.ln.lnaddr
	c.remoteAddr = c.conn.RemoteAddr()
	el.calibrateCallback(el, 1)

	out, action := el.eventHandler.OnOpened(c)
	if out != nil {
		el.eventHandler.PreWrite()
		_, _ = c.conn.Write(out)
	}
	if el.svr.opts.TCPKeepAlive > 0 {
		if c, ok := c.conn.(*net.TCPConn); ok {
			_ = c.SetKeepAlive(true)
			_ = c.SetKeepAlivePeriod(el.svr.opts.TCPKeepAlive)
		}
	}
	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(ti *tcpIn) (err error) {
	c := ti.c
	c.buffer = ti.in

	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			outFrame, _ := el.codec.Encode(c, out)
			el.eventHandler.PreWrite()
			_, err = c.conn.Write(outFrame)
		}
		switch action {
		case None:
		case Close:
			return el.loopCloseConn(c)
		case Shutdown:
			return errServerShutdown
		}
		if err != nil {
			return el.loopError(c, err)
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
				err = errServerShutdown
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
	if e = c.conn.Close(); e == nil {
		delete(el.connections, c)
		el.calibrateCallback(el, -1)
		switch el.eventHandler.OnClosed(c, err) {
		case Shutdown:
			return errServerShutdown
		}
		c.releaseTCP()
	} else {
		el.svr.logger.Printf("failed to close connection:%s, error:%v\n", c.remoteAddr.String(), e)
	}
	return
}

func (el *eventloop) loopWake(c *stdConn) error {
	//if co, ok := el.connections[c]; !ok || co != c {
	//	return nil // ignore stale wakes.
	//}
	out, action := el.eventHandler.React(nil, c)
	if out != nil {
		frame, _ := el.codec.Encode(c, out)
		_, _ = c.conn.Write(frame)
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
		return errServerShutdown
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
		return errServerShutdown
	}
	c.releaseUDP()
	return nil
}
