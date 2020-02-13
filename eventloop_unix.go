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

type eventloop struct {
	idx          int             // loop index in the server loops list
	svr          *server         // server in loop
	codec        ICodec          // codec for TCP
	packet       []byte          // read packet buffer
	poller       *netpoll.Poller // epoll or kqueue
	connections  map[int]*conn   // loop connections fd -> conn
	eventHandler EventHandler    // user eventHandler
}

func (el *eventloop) loopRun() {
	defer func() {
		if el.idx == 0 && el.svr.opts.Ticker {
			close(el.svr.ticktock)
		}
		el.svr.signalShutdown()
	}()

	if el.idx == 0 && el.svr.opts.Ticker {
		go el.loopTicker()
	}

	sniffError(el.poller.Polling(el.handleEvent))
}

func (el *eventloop) loopAccept(fd int) error {
	if fd == el.svr.ln.fd {
		if el.svr.ln.pconn != nil {
			return el.loopReadUDP(fd)
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
		c := newTCPConn(nfd, el, sa)
		if err = el.poller.AddRead(c.fd); err == nil {
			el.connections[c.fd] = c
			return el.loopOpen(c)
		}
		return err
	}
	return nil
}

func (el *eventloop) loopOpen(c *conn) error {
	c.opened = true
	c.localAddr = el.svr.ln.lnaddr
	c.remoteAddr = netpoll.SockaddrToTCPOrUnixAddr(c.sa)
	out, action := el.eventHandler.OnOpened(c)
	if el.svr.opts.TCPKeepAlive > 0 {
		if _, ok := el.svr.ln.ln.(*net.TCPListener); ok {
			_ = netpoll.SetKeepAlive(c.fd, int(el.svr.opts.TCPKeepAlive/time.Second))
		}
	}
	if out != nil {
		c.open(out)
	}

	if !c.outboundBuffer.IsEmpty() {
		_ = el.poller.AddWrite(c.fd)
	}

	return el.handleAction(c, action)
}

func (el *eventloop) loopRead(c *conn) error {
	n, err := unix.Read(c.fd, el.packet)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return el.loopCloseConn(c, err)
	}
	c.buffer = el.packet[:n]

	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			outFrame, _ := el.codec.Encode(c, out)
			c.write(outFrame)
		}
		switch action {
		case None:
		case Close:
			_ = el.loopWrite(c)
			return el.loopCloseConn(c, nil)
		case Shutdown:
			_ = el.loopWrite(c)
			return errServerShutdown
		}
		if !c.opened {
			return nil
		}
	}
	_, _ = c.inboundBuffer.Write(c.buffer)

	return nil
}

func (el *eventloop) loopWrite(c *conn) error {
	el.eventHandler.PreWrite()

	head, tail := c.outboundBuffer.LazyReadAll()
	n, err := unix.Write(c.fd, head)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return el.loopCloseConn(c, err)
	}
	c.outboundBuffer.Shift(n)

	if len(head) == n && tail != nil {
		n, err = unix.Write(c.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return el.loopCloseConn(c, err)
		}
		c.outboundBuffer.Shift(n)
	}

	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.fd)
	}
	return nil
}

func (el *eventloop) loopCloseConn(c *conn, err error) error {
	if el.poller.Delete(c.fd) == nil && unix.Close(c.fd) == nil {
		delete(el.connections, c.fd)
		switch el.eventHandler.OnClosed(c, err) {
		case Shutdown:
			return errServerShutdown
		}
		c.releaseTCP()
	}
	return nil
}

func (el *eventloop) loopWake(c *conn) error {
	//if co, ok := el.connections[c.fd]; !ok || co != c {
	//	return nil // ignore stale wakes.
	//}
	out, action := el.eventHandler.React(nil, c)
	if out != nil {
		frame, _ := el.codec.Encode(c, out)
		c.write(frame)
	}
	return el.handleAction(c, action)
}

func (el *eventloop) loopTicker() {
	var (
		delay time.Duration
		open  bool
	)
	for {
		sniffError(el.poller.Trigger(func() (err error) {
			delay, action := el.eventHandler.Tick()
			el.svr.ticktock <- delay
			switch action {
			case None:
			case Shutdown:
				err = errServerShutdown
			}
			return
		}))
		if delay, open = <-el.svr.ticktock; open {
			time.Sleep(delay)
		} else {
			break
		}
	}
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		_ = el.loopWrite(c)
		return el.loopCloseConn(c, nil)
	case Shutdown:
		_ = el.loopWrite(c)
		return errServerShutdown
	default:
		return nil
	}
}

func (el *eventloop) loopReadUDP(fd int) error {
	n, sa, err := unix.Recvfrom(fd, el.packet, 0)
	if err != nil || n == 0 {
		return nil
	}
	c := newUDPConn(fd, el, sa)
	out, action := el.eventHandler.React(el.packet[:n], c)
	if out != nil {
		el.eventHandler.PreWrite()
		c.sendTo(out)
	}
	switch action {
	case Shutdown:
		return errServerShutdown
	}
	c.releaseUDP()
	return nil
}
