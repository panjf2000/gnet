// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type loop struct {
	ch          chan interface{}  // command channel
	idx         int               // loop index
	svr         *server           // server in loop
	connections map[*stdConn]bool // track all the sockets bound to this loop
}

func (lp *loop) loopRun() {
	var err error
	defer func() {
		if lp.idx == 0 && lp.svr.opts.Ticker {
			close(lp.svr.ticktock)
		}
		lp.svr.signalShutdown(err)
		lp.svr.loopWG.Done()
		lp.loopEgress()
		lp.svr.loopWG.Done()
	}()
	if lp.idx == 0 && lp.svr.opts.Ticker {
		go lp.loopTicker()
	}
	for v := range lp.ch {
		switch v := v.(type) {
		case error:
			err = v
		case *stdConn:
			err = lp.loopAccept(v)
		case *tcpIn:
			err = lp.loopRead(v)
		case *udpIn:
			err = lp.loopReadUDP(v.c)
		case *stderr:
			err = lp.loopError(v.c, v.err)
		case wakeReq:
			err = lp.loopWake(v.c)
		case func() error:
			err = v()
		}
		if err != nil {
			return
		}
	}
}

func (lp *loop) loopAccept(c *stdConn) error {
	lp.connections[c] = true
	c.localAddr = lp.svr.ln.lnaddr
	c.remoteAddr = c.conn.RemoteAddr()

	out, action := lp.svr.eventHandler.OnOpened(c)
	if out != nil {
		lp.svr.eventHandler.PreWrite()
		_, _ = c.conn.Write(out)
	}
	if lp.svr.opts.TCPKeepAlive > 0 {
		if c, ok := c.conn.(*net.TCPConn); ok {
			_ = c.SetKeepAlive(true)
			_ = c.SetKeepAlivePeriod(lp.svr.opts.TCPKeepAlive)
		}
	}
	switch action {
	case Shutdown:
		return errClosing
	case Close:
		return lp.loopClose(c)
	}
	return nil
}

func (lp *loop) loopRead(ti *tcpIn) error {
	c := ti.c
	c.cache = ti.in
loopReact:
	out, action := lp.svr.eventHandler.React(c)
	if len(out) != 0 {
		lp.svr.eventHandler.PreWrite()
		if frame, err := lp.svr.codec.Encode(out); err == nil {
			_, _ = c.conn.Write(frame)
		}
		goto loopReact
	}
	_, _ = c.inboundBuffer.Write(c.cache)
	switch action {
	case Shutdown:
		return errClosing
	case Close:
		return lp.loopClose(c)
	}
	return nil
}

func (lp *loop) loopClose(c *stdConn) error {
	atomic.StoreInt32(&c.done, 1)
	_ = c.conn.SetReadDeadline(time.Now())
	return nil
}

func (lp *loop) loopEgress() {
	var closed bool
	for v := range lp.ch {
		switch v := v.(type) {
		case error:
			if v == errCloseConns {
				closed = true
				for c := range lp.connections {
					_ = lp.loopClose(c)
				}
			}
		case *stderr:
			_ = lp.loopError(v.c, v.err)
		}
		if len(lp.connections) == 0 && closed {
			break
		}
	}
}

func (lp *loop) loopTicker() {
	for {
		lp.ch <- func() (err error) {
			delay, action := lp.svr.eventHandler.Tick()
			lp.svr.ticktock <- delay
			switch action {
			case Shutdown:
				err = errClosing
			}
			return
		}
		delay, ok := <-lp.svr.ticktock
		if !ok {
			break
		}
		time.Sleep(delay)
	}
}

func (lp *loop) loopError(c *stdConn, err error) (e error) {
	if e = c.conn.Close(); e == nil {
		delete(lp.connections, c)
		switch atomic.LoadInt32(&c.done) {
		case 0: // read error
			if err != io.EOF {
				log.Printf("socket: %s with err: %v\n", c.remoteAddr.String(), err)
			}
		case 1: // closed
			log.Printf("socket: %s has been closed by client\n", c.remoteAddr.String())
		}
		switch lp.svr.eventHandler.OnClosed(c, err) {
		case Shutdown:
			return errClosing
		}
		c.release()
	}
	return
}

func (lp *loop) loopWake(c *stdConn) error {
	if _, ok := lp.connections[c]; !ok {
		return nil // ignore stale wakes.
	}
	out, action := lp.svr.eventHandler.React(c)
	if out != nil {
		_, _ = c.conn.Write(out)
	}
	switch action {
	case Shutdown:
		return errClosing
	case Close:
		return lp.loopClose(c)
	}
	return nil
}

func (lp *loop) loopReadUDP(c *stdConn) error {
	out, action := lp.svr.eventHandler.React(c)
	if out != nil {
		lp.svr.eventHandler.PreWrite()
		_, _ = lp.svr.ln.pconn.WriteTo(out, c.remoteAddr)
	}
	switch action {
	case Shutdown:
		return errClosing
	}
	c.release()
	return nil
}
