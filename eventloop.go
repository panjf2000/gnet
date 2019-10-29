// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"net"
	"time"

	"github.com/panjf2000/gnet/netpoll"
	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

type loop struct {
	idx         int             // loop index in the server loops list
	svr         *server         // server in loop
	packet      []byte          // read packet buffer
	poller      *netpoll.Poller // epoll or kqueue
	connections map[int]*conn   // loop connections fd -> conn
}

func (lp *loop) loopRun() {
	defer lp.svr.signalShutdown()

	if lp.idx == 0 && lp.svr.opts.Ticker {
		go lp.loopTicker()
	}

	_ = lp.poller.Polling(lp.handleEvent)
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
		c := newConn(nfd, lp, sa)
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
	out, action := lp.svr.eventHandler.OnOpened(c)
	c.action = action
	if lp.svr.opts.TCPKeepAlive > 0 {
		if _, ok := lp.svr.ln.ln.(*net.TCPListener); ok {
			sniffError(netpoll.SetKeepAlive(c.fd, int(lp.svr.opts.TCPKeepAlive/time.Second)))
		}
	}

	if out != nil {
		c.open(out)
	}
	if !c.outboundBuffer.IsEmpty() {
		_ = lp.poller.AddWrite(c.fd)
	}
	return lp.handleAction(c)
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
	out, action := lp.svr.eventHandler.React(c)

	if out != nil {
		c.write(out)
	}
	_, _ = c.inboundBuffer.Write(c.cache)

	c.action = action
	return lp.handleAction(c)
}

func (lp *loop) loopOut(c *conn) error {
	lp.svr.eventHandler.PreWrite()

	top, tail := c.outboundBuffer.PreReadAll()
	n, err := unix.Write(c.fd, top)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return lp.loopCloseConn(c, err)
	}
	c.outboundBuffer.Shift(n)

	if len(top) == n && tail != nil {
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
		switch lp.svr.eventHandler.OnClosed(c, err) {
		case Shutdown:
			return errShutdown
		}
		c.reset()
	}
	return nil
}

func (lp *loop) loopWake(c *conn) error {
	if co, ok := lp.connections[c.fd]; !ok || co != c {
		return nil // ignore stale wakes.
	}
	out, action := lp.svr.eventHandler.React(c)
	c.action = action
	if out != nil {
		c.write(out)
	}
	return lp.handleAction(c)
}

func (lp *loop) loopTicker() {
	for {
		if err := lp.poller.Trigger(func() (err error) {
			delay, action := lp.svr.eventHandler.Tick()
			lp.svr.tch <- delay
			switch action {
			case None:
			case Shutdown:
				err = errShutdown
			}
			return
		}); err != nil {
			break
		}
		time.Sleep(<-lp.svr.tch)
	}
}

func (lp *loop) handleAction(c *conn) error {
	switch c.action {
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
	var sa6 unix.SockaddrInet6
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		sa6.ZoneId = 0
		sa6.Port = sa.Port
		for i := 0; i < 12; i++ {
			sa6.Addr[i] = 0
		}
		sa6.Addr[12] = sa.Addr[0]
		sa6.Addr[13] = sa.Addr[1]
		sa6.Addr[14] = sa.Addr[2]
		sa6.Addr[15] = sa.Addr[3]
	case *unix.SockaddrInet6:
		sa6 = *sa
	}
	c := &conn{
		fd:            fd,
		localAddr:     lp.svr.ln.lnaddr,
		remoteAddr:    netpoll.SockaddrToUDPAddr(&sa6),
		inboundBuffer: lp.svr.bytesPool.Get().(*ringbuffer.RingBuffer),
	}
	c.cache = lp.packet[:n]
	out, action := lp.svr.eventHandler.React(c)
	if out != nil {
		lp.svr.eventHandler.PreWrite()
		c.sendTo(out, sa)
	}
	switch action {
	case Shutdown:
		return errShutdown
	}
	lp.svr.bytesPool.Put(c.inboundBuffer)
	return nil
}
