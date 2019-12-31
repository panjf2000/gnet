// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"net"
	"time"

	"github.com/panjf2000/gnet/internal/netpoll"
	"golang.org/x/sys/unix"
)

type loop struct {
	idx          int             // loop index in the server loops list
	svr          *server         // server in loop
	codec        ICodec          // codec for TCP
	packet       []byte          // read packet buffer
	poller       *netpoll.Poller // epoll or kqueue
	connections  map[int]*conn   // loop connections fd -> conn
	eventHandler EventHandler    // user eventHandler
}

func (lp *loop) loopRun() {
	defer func() {
		if lp.idx == 0 && lp.svr.opts.Ticker {
			close(lp.svr.ticktock)
		}
		lp.svr.signalShutdown()
	}()

	if lp.idx == 0 && lp.svr.opts.Ticker {
		go lp.loopTicker()
	}

	sniffError(lp.poller.Polling(lp.handleEvent))
}

func (lp *loop) loopAccept(fd int) error {
	if fd == lp.svr.ln.fd {
		if lp.svr.ln.pconn != nil {
			return lp.loopUDPIn(fd)
		}
		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return err
		}
		if err := unix.SetNonblock(nfd, true); err != nil {
			return err
		}
		c := newTCPConn(nfd, lp, sa)
		if err = lp.poller.AddReadWrite(c.fd); err == nil {
			lp.connections[c.fd] = c
		} else {
			return err
		}
	}
	return nil
}

func (lp *loop) loopOpen(c *conn) error {
	c.opened = true
	c.localAddr = lp.svr.ln.lnaddr
	c.remoteAddr = netpoll.SockaddrToTCPOrUnixAddr(c.sa)
	out, action := lp.eventHandler.OnOpened(c)
	if lp.svr.opts.TCPKeepAlive > 0 {
		if _, ok := lp.svr.ln.ln.(*net.TCPListener); ok {
			_ = netpoll.SetKeepAlive(c.fd, int(lp.svr.opts.TCPKeepAlive/time.Second))
		}
	}
	if out != nil {
		c.open(out)
	}

	if !c.outboundBuffer.IsEmpty() {
		_ = lp.poller.AddWrite(c.fd)
	}

	return lp.handleAction(c, action)
}

func (lp *loop) loopIn(c *conn) error {
	n, err := unix.Read(c.fd, lp.packet)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return lp.loopCloseConn(c, err)
	}
	c.cache = lp.packet[:n]

	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := lp.eventHandler.React(inFrame, c)
		if out != nil {
			if outFrame, err := lp.codec.Encode(c, out); err == nil {
				c.write(outFrame)
			}
		}

		switch action {
		case None:
		case Close:
			return lp.loopCloseConn(c, nil)
		case Shutdown:
			return errShutdown
		}
	}
	_, _ = c.inboundBuffer.Write(c.cache)

	return nil
}

func (lp *loop) loopOut(c *conn) error {
	lp.eventHandler.PreWrite()

	head, tail := c.outboundBuffer.LazyReadAll()
	n, err := unix.Write(c.fd, head)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return lp.loopCloseConn(c, err)
	}
	c.outboundBuffer.Shift(n)

	if len(head) == n && tail != nil {
		n, err = unix.Write(c.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return lp.loopCloseConn(c, err)
		}
		c.outboundBuffer.Shift(n)
	}

	if c.outboundBuffer.IsEmpty() {
		_ = lp.poller.ModRead(c.fd)
	}
	return nil
}

func (lp *loop) loopCloseConn(c *conn, err error) error {
	if lp.poller.Delete(c.fd) == nil && unix.Close(c.fd) == nil {
		delete(lp.connections, c.fd)
		switch lp.eventHandler.OnClosed(c, err) {
		case Shutdown:
			return errShutdown
		}
		c.releaseTCP()
	}
	return nil
}

func (lp *loop) loopWake(c *conn) error {
	if co, ok := lp.connections[c.fd]; !ok || co != c {
		return nil // ignore stale wakes.
	}
	out, action := lp.eventHandler.React(nil, c)
	if out != nil {
		c.write(out)
	}
	return lp.handleAction(c, action)
}

func (lp *loop) loopTicker() {
	var (
		delay time.Duration
		open  bool
	)
	for {
		if err := lp.poller.Trigger(func() (err error) {
			delay, action := lp.eventHandler.Tick()
			lp.svr.ticktock <- delay
			switch action {
			case None:
			case Shutdown:
				err = errShutdown
			}
			return
		}); err != nil {
			break
		}
		if delay, open = <-lp.svr.ticktock; open {
			time.Sleep(delay)
		} else {
			break
		}
	}
}

func (lp *loop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return lp.loopCloseConn(c, nil)
	case Shutdown:
		return errShutdown
	default:
		return nil
	}
}

func (lp *loop) loopUDPIn(fd int) error {
	n, sa, err := unix.Recvfrom(fd, lp.packet, 0)
	if err != nil || n == 0 {
		return nil
	}
	c := newUDPConn(fd, lp, sa)
	out, action := lp.eventHandler.React(lp.packet[:n], c)
	if out != nil {
		lp.eventHandler.PreWrite()
		c.sendTo(out)
	}
	switch action {
	case Shutdown:
		return errShutdown
	}
	c.releaseUDP()
	return nil
}
