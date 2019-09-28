// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"net"
	"time"

	"github.com/panjf2000/gnet/internal"
	"github.com/panjf2000/gnet/netpoll"
	"github.com/panjf2000/gnet/ringbuffer"
	"golang.org/x/sys/unix"
)

type loop struct {
	idx         int             // loop index in the server loops list
	poller      *netpoll.Poller // epoll or kqueue
	packet      []byte          // read packet buffer
	connections map[int]*conn   // loop connections fd -> conn
	svr         *server
}

func (l *loop) loopRun() {
	defer l.svr.signalShutdown()

	if l.idx == 0 && l.svr.events.Tick != nil {
		go l.loopTicker()
	}

	_ = l.poller.Polling(func(fd int, job internal.Job) error {
		if fd == 0 {
			return job()
		}
		if c, ok := l.connections[fd]; ok {
			switch {
			case !c.opened:
				return l.loopOpened(c)
			case c.outBuf.Length() > 0:
				return l.loopWrite(c)
			default:
				return l.loopRead(c)
			}
		} else {
			return l.loopAccept(fd)
		}
	})
}

func (l *loop) loopAccept(fd int) error {
	if fd == l.svr.ln.fd {
		if l.svr.ln.pconn != nil {
			return l.loopUDPRead(fd)
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
		conn := &conn{fd: nfd,
			sa:     sa,
			inBuf:  ringbuffer.New(connRingBufferSize),
			outBuf: ringbuffer.New(connRingBufferSize),
			loop:   l,
		}
		if err = l.poller.AddReadWrite(conn.fd); err == nil {
			l.connections[conn.fd] = conn
		} else {
			return err
		}
	}
	return nil
}

func (l *loop) loopOpened(conn *conn) error {
	conn.opened = true
	conn.localAddr = l.svr.ln.lnaddr
	conn.remoteAddr = netpoll.SockaddrToTCPOrUnixAddr(conn.sa)
	if l.svr.events.OnOpened != nil {
		out, opts, action := l.svr.events.OnOpened(conn)
		conn.action = action
		if opts.TCPKeepAlive > 0 {
			if _, ok := l.svr.ln.ln.(*net.TCPListener); ok {
				sniffError(netpoll.SetKeepAlive(conn.fd, int(opts.TCPKeepAlive/time.Second)))
			}
		}

		if len(out) > 0 {
			conn.open(out)
		}
	}
	if conn.outBuf.Length() != 0 {
		_ = l.poller.AddWrite(conn.fd)
	}
	return l.handleAction(conn)
}

func (l *loop) loopRead(conn *conn) error {
	n, err := unix.Read(conn.fd, l.packet)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return l.loopCloseConn(conn, err)
	}
	conn.extra = l.packet[:n]
	out, action := l.svr.events.React(conn)
	conn.action = action
	if len(out) > 0 {
		conn.write(out)
	} else if action != DataRead {
		_, _ = conn.inBuf.Write(conn.extra)
	}
	return l.handleAction(conn)
}

func (l *loop) loopWrite(conn *conn) error {
	if l.svr.events.PreWrite != nil {
		l.svr.events.PreWrite()
	}

	top, tail := conn.outBuf.PreReadAll()
	n, err := unix.Write(conn.fd, top)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return l.loopCloseConn(conn, err)
	}
	conn.outBuf.Advance(n)
	if len(top) == n && tail != nil {
		n, err = unix.Write(conn.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return l.loopCloseConn(conn, err)
		}
		conn.outBuf.Advance(n)
	}

	if conn.outBuf.Length() == 0 {
		_ = l.poller.ModRead(conn.fd)
	}
	return nil
}

func (l *loop) loopCloseConn(conn *conn, err error) error {
	if err := l.poller.Delete(conn.fd); err == nil {
		delete(l.connections, conn.fd)
		_ = unix.Close(conn.fd)
	}

	if l.svr.events.OnClosed != nil {
		switch l.svr.events.OnClosed(conn, err) {
		case None:
		case Shutdown:
			return ErrClosing
		}
	}
	return nil
}

//func (l *loop) loopWake(conn *conn) error {
//	out, action := l.svr.events.React(conn)
//	conn.action = action
//	if len(out) > 0 {
//		conn.write(out)
//	}
//	return l.handleAction(conn)
//}

//func (l *loop) loopNote(job internal.Job) error {
//
//	var err error
//	switch v := job.(type) {
//	case *conn:
//		l.connections[v.fd] = v
//		l.poller.AddRead(v.fd)
//		return nil
//	case func() error:
//		return v()
//	case time.Duration:
//		delay, action := l.svr.events.Tick()
//		switch action {
//		case None:
//		case Shutdown:
//			err = ErrClosing
//		}
//		l.svr.tch <- delay
//	case error: // shutdown
//		err = v
//		//case *conn:
//		//	// Wake called for connection
//		//	if val, ok := l.connections[v.fd]; !ok || val != v {
//		//		return nil // ignore stale wakes
//		//	}
//		//	return l.loopWake(v)
//	}
//	return err
//}

func (l *loop) loopTicker() {
	for {
		if err := l.poller.Trigger(func() (err error) {
			delay, action := l.svr.events.Tick()
			l.svr.tch <- delay
			switch action {
			case None:
			case Shutdown:
				err = ErrClosing
			}
			return err
		}); err != nil {
			break
		}
		time.Sleep(<-l.svr.tch)
	}
}

func (l *loop) handleAction(conn *conn) error {
	switch conn.action {
	case None:
		return nil
	case Close:
		return l.loopCloseConn(conn, nil)
	case Shutdown:
		return ErrClosing
	default:
		return nil
	}
}

func (l *loop) loopUDPRead(fd int) error {
	n, sa, err := unix.Recvfrom(fd, l.packet, 0)
	if err != nil || n == 0 {
		return nil
	}
	if l.svr.events.React != nil {
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
		conn := &conn{
			localAddr:  l.svr.ln.lnaddr,
			remoteAddr: netpoll.SockaddrToUDPAddr(&sa6),
			inBuf:      ringbuffer.New(connRingBufferSize),
		}
		_, _ = conn.inBuf.Write(l.packet[:n])
		out, action := l.svr.events.React(conn)
		if len(out) > 0 {
			if l.svr.events.PreWrite != nil {
				l.svr.events.PreWrite()
			}
			sniffError(unix.Sendto(fd, out, 0, sa))
		}
		switch action {
		case Shutdown:
			return ErrClosing
		}
	}
	return nil
}
