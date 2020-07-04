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

// +build linux freebsd dragonfly darwin

package gnet

import (
	"os"
	"time"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/netpoll"
	"golang.org/x/sys/unix"
)

type eventloop struct {
	ln                *listener               // listener
	idx               int                     // loop index in the server loops list
	svr               *server                 // server in loop
	codec             ICodec                  // codec for TCP
	packet            []byte                  // read packet buffer
	poller            *netpoll.Poller         // epoll or kqueue
	connCount         int32                   // number of active connections in event-loop
	connections       map[int]*conn           // loop connections fd -> conn
	eventHandler      EventHandler            // user eventHandler
	calibrateCallback func(*eventloop, int32) // callback func for re-adjusting connCount
}

func (el *eventloop) closeAllConns() {
	// Close loops and all outstanding connections
	for _, c := range el.connections {
		_ = el.loopCloseConn(c, nil)
	}
}

func (el *eventloop) loopRun() {
	defer func() {
		el.closeAllConns()
		el.ln.close()
		if el.idx == 0 && el.svr.opts.Ticker {
			close(el.svr.ticktock)
		}
		el.svr.signalShutdown()
	}()

	if el.idx == 0 && el.svr.opts.Ticker {
		go el.loopTicker()
	}

	switch err := el.poller.Polling(el.handleEvent); err {
	case errors.ErrServerShutdown:
		el.svr.logger.Infof("Event-loop(%d) is exiting normally on the signal error: %v", el.idx, err)
	default:
		el.svr.logger.Errorf("Event-loop(%d) is exiting due to an unexpected error: %v", el.idx, err)

	}
}

func (el *eventloop) loopAccept(fd int) error {
	if fd == el.ln.fd {
		if el.ln.network == "udp" {
			return el.loopReadUDP(fd)
		}

		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return os.NewSyscallError("accept", err)
		}
		if err = unix.SetNonblock(nfd, true); err != nil {
			return os.NewSyscallError("fcntl nonblock", err)
		}
		c := newTCPConn(nfd, el, sa)
		if err = el.poller.AddRead(c.fd); err == nil {
			el.connections[c.fd] = c
			el.calibrateCallback(el, 1)
			return el.loopOpen(c)
		}
		return err
	}

	return nil
}

func (el *eventloop) loopOpen(c *conn) error {
	c.opened = true
	c.localAddr = el.ln.lnaddr
	c.remoteAddr = netpoll.SockaddrToTCPOrUnixAddr(c.sa)
	if el.svr.opts.TCPKeepAlive > 0 {
		if proto := el.ln.network; proto == "tcp" || proto == "unix" {
			_ = netpoll.SetKeepAlive(c.fd, int(el.svr.opts.TCPKeepAlive/time.Second))
		}
	}

	out, action := el.eventHandler.OnOpened(c)
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
		return el.loopCloseConn(c, os.NewSyscallError("read", err))
	}
	c.buffer = el.packet[:n]

	for inFrame, _ := c.read(); inFrame != nil; inFrame, _ = c.read() {
		out, action := el.eventHandler.React(inFrame, c)
		if out != nil {
			outFrame, _ := el.codec.Encode(c, out)
			el.eventHandler.PreWrite()
			c.write(outFrame)
		}
		switch action {
		case None:
		case Close:
			return el.loopCloseConn(c, nil)
		case Shutdown:
			return errors.ErrServerShutdown
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
		return el.loopCloseConn(c, os.NewSyscallError("write", err))
	}
	c.outboundBuffer.Shift(n)

	if len(head) == n && tail != nil {
		n, err = unix.Write(c.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return el.loopCloseConn(c, os.NewSyscallError("write", err))
		}
		c.outboundBuffer.Shift(n)
	}

	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.fd)
	}

	return nil
}

func (el *eventloop) loopCloseConn(c *conn, err error) error {
	if !c.outboundBuffer.IsEmpty() && err == nil {
		_ = el.loopWrite(c)
	}

	if !c.opened {
		el.svr.logger.Debugf("The fd=%d in event-loop(%d) is already closed, skipping it", c.fd, el.idx)
		return nil
	}

	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 == nil && err1 == nil {
		delete(el.connections, c.fd)
		el.calibrateCallback(el, -1)
		if el.eventHandler.OnClosed(c, err) == Shutdown {
			return errors.ErrServerShutdown
		}
		c.releaseTCP()
	} else {
		if err0 != nil {
			el.svr.logger.Warnf("Failed to delete fd=%d from poller in event-loop(%d), %v", c.fd, el.idx, err0)
		}
		if err1 != nil {
			el.svr.logger.Warnf("Failed to close fd=%d in event-loop(%d), %v",
				c.fd, el.idx, os.NewSyscallError("close", err1))
		}
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
		err   error
	)
	for {
		err = el.poller.Trigger(func() (err error) {
			delay, action := el.eventHandler.Tick()
			el.svr.ticktock <- delay
			switch action {
			case None:
			case Shutdown:
				err = errors.ErrServerShutdown
			}
			return
		})
		if err != nil {
			el.svr.logger.Errorf("Failed to awake poller in event-loop(%d), error:%v, stopping ticker", el.idx, err)
			break
		}
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
		return el.loopCloseConn(c, nil)
	case Shutdown:
		return errors.ErrServerShutdown
	default:
		return nil
	}
}

func (el *eventloop) loopReadUDP(fd int) error {
	n, sa, err := unix.Recvfrom(fd, el.packet, 0)
	if err != nil || n == 0 {
		if err != nil && err != unix.EAGAIN {
			el.svr.logger.Warnf("Failed to read UDP packet from fd=%d in event-loop(%d), %v",
				fd, el.idx, os.NewSyscallError("recvfrom", err))
		}
		return nil
	}

	c := newUDPConn(fd, el, sa)
	out, action := el.eventHandler.React(el.packet[:n], c)
	if out != nil {
		el.eventHandler.PreWrite()
		_ = c.sendTo(out)
	}
	if action == Shutdown {
		return errors.ErrServerShutdown
	}
	c.releaseUDP()

	return nil
}
