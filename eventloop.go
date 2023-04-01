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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/io"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	gerrors "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type eventloop struct {
	ln           *listener       // listener
	idx          int             // loop index in the engine loops list
	cache        bytes.Buffer    // temporary buffer for scattered bytes
	engine       *engine         // engine in loop
	poller       *netpoll.Poller // epoll or kqueue
	buffer       []byte          // read packet buffer whose capacity is set by user, default value is 64KB
	connCount    int32           // number of active connections in event-loop
	udpSockets   map[int]*conn   // client-side UDP socket map: fd -> conn
	connections  map[int]*conn   // TCP connection map: fd -> conn
	eventHandler EventHandler    // user eventHandler
}

func (el *eventloop) getLogger() logging.Logger {
	return el.engine.opts.Logger
}

func (el *eventloop) addConn(delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}

func (el *eventloop) loadConn() int32 {
	return atomic.LoadInt32(&el.connCount)
}

func (el *eventloop) closeAllSockets() {
	// Close loops and all outstanding connections
	for _, c := range el.connections {
		_ = el.closeConn(c, nil)
	}
	for _, c := range el.udpSockets {
		_ = el.closeConn(c, nil)
	}
}

func (el *eventloop) register(itf interface{}) error {
	c := itf.(*conn)
	if c.pollAttachment == nil { // UDP socket
		c.pollAttachment = netpoll.GetPollAttachment()
		c.pollAttachment.FD = c.fd
		c.pollAttachment.Callback = el.readUDP
		if err := el.poller.AddRead(c.pollAttachment); err != nil {
			_ = unix.Close(c.fd)
			c.releaseUDP()
			return err
		}
		el.udpSockets[c.fd] = c
		return nil
	}
	if err := el.poller.AddRead(c.pollAttachment); err != nil {
		_ = unix.Close(c.fd)
		c.releaseTCP()
		return err
	}
	el.connections[c.fd] = c
	return el.open(c)
}

func (el *eventloop) open(c *conn) error {
	c.opened = true
	el.addConn(1)

	out, action := el.eventHandler.OnOpen(c)
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() {
		if err := el.poller.AddWrite(c.pollAttachment); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) readTLS(c *conn) error {
	// Since the el.Buffer may contain multiple TLS record,
	// we process one TLS record in each iteration until no more
	// TLS records are available
	for {
		if err := c.tlsconn.ReadFrame(); err != nil {
			// Receive error unix.EAGAIN, wait for the next round
			if err == unix.EAGAIN {
				c.tlsconn.DataDone()
				return nil
			}
			// If err is io.EOF, it can either the data is drained,
			// receives a close notify from the client.
			return el.closeConn(c, os.NewSyscallError("TLS read", err))
		}

		// load all decrypted data and make it ready for gnet to use
		c.buffer = c.tlsconn.Data()

		action := el.eventHandler.OnTraffic(c)
		switch action {
		case None:
		case Close:
			// tls data will be cleaned up in el.closeConn()
			return el.closeConn(c, nil)
		case Shutdown:
			c.tlsconn.DataDone()
			return gerrors.ErrEngineShutdown
		}
		_, _ = c.inboundBuffer.Write(c.buffer)
		c.buffer = c.buffer[:0]

		// all available TLS records are processed
		if !c.tlsconn.IsRecordCompleted(c.tlsconn.RawInputData()) {
			c.tlsconn.DataDone()
			return nil
		}
	}
}

func (el *eventloop) read(c *conn) error {
	// detected whether kernel TLS RX is enabled
	// This only happens after TLS handshake is completed.
	// Therefore, no need to call c.tlsconn.HandshakeComplete().
	if c.tlsconn != nil && c.tlsconn.IsKTLSRXEnabled() {
		// attach the gnet eventloop.buffer to tlsconn.rawInput.
		// So, KTLS can decrypt the data directly to the buffer without memory allocation.
		// Since data is read through KTLS, there is no need to call unix.read(c.fd, el.buffer)
		c.tlsconn.RawInputSet(el.buffer) //nolint:errcheck
		return el.readTLS(c)
	}

	n, err := unix.Read(c.fd, el.buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = unix.ECONNRESET
		}
		return el.closeConn(c, os.NewSyscallError("read", err))
	}

	if c.tlsconn != nil {
		// attach the gnet eventloop.buffer to tlsconn.rawInput.
		c.tlsconn.RawInputSet(el.buffer[:n]) //nolint:errcheck
		if !c.tlsconn.HandshakeComplete() {
			// 先判断是否足够一条消息
			data := c.tlsconn.RawInputData()
			if !c.tlsconn.IsRecordCompleted(data) {
				c.tlsconn.DataDone()
				return nil
			}
			if err = c.tlsconn.Handshake(); err != nil {
				// closeConn will cleanup the TLS data at the end,
				// so no need to call tlsconn.DataDone()
				return el.closeConn(c, os.NewSyscallError("TLS handshake", err))
			}
			if !c.tlsconn.HandshakeComplete() || len(c.tlsconn.RawInputData()) == 0 { // 握手没成功，或者握手成功，但是没有数据黏包了
				c.tlsconn.DataDone()
				return nil
			}
		}
		return el.readTLS(c)
	}

	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	switch action {
	case None:
	case Close:
		return el.closeConn(c, nil)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	c.buffer = c.buffer[:0]
	return nil
}

const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	iov := c.outboundBuffer.Peek(-1)
	var (
		n   int
		err error
	)
	if len(iov) > 1 {
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err = io.Writev(c.fd, iov)
	} else {
		n, err = unix.Write(c.fd, iov[0])
	}
	_, _ = c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.closeConn(c, os.NewSyscallError("write", err))
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.pollAttachment)
	}

	return nil
}

func (el *eventloop) closeConn(c *conn, err error) (rerr error) {
	if addr := c.localAddr; addr != nil && strings.HasPrefix(c.localAddr.Network(), "udp") {
		rerr = el.poller.Delete(c.fd)
		if c.fd != el.ln.fd {
			rerr = unix.Close(c.fd)
			delete(el.udpSockets, c.fd)
		}
		if el.eventHandler.OnClose(c, err) == Shutdown {
			return gerrors.ErrEngineShutdown
		}
		c.releaseUDP()
		return
	}

	if !c.opened {
		return
	}

	// close the TLS connection by sending the alert
	if c.tlsconn != nil {
		// close the TLS connection, which will send a close notify to the client
		c.tlsconn.Close()
		// Make sure all memory requested from the pool is returned.
		c.tlsconn.DataCleanUpAfterClose()
		c.tlsconn = nil
		// TODO: create a sync.pool to manage the TLS connection
	}

	// Send residual data in buffer back to the peer before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		for !c.outboundBuffer.IsEmpty() {
			iov := c.outboundBuffer.Peek(0)
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			if n, e := io.Writev(c.fd, iov); e != nil {
				el.getLogger().Warnf("closeConn: error occurs when sending data back to peer, %v", e)
				break
			} else {
				_, _ = c.outboundBuffer.Discard(n)
			}
		}
	}

	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 != nil {
		rerr = fmt.Errorf("failed to delete fd=%d from poller in event-loop(%d): %v", c.fd, el.idx, err0)
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d in event-loop(%d): %v", c.fd, el.idx, os.NewSyscallError("close", err1))
		if rerr != nil {
			rerr = errors.New(rerr.Error() + " & " + err1.Error())
		} else {
			rerr = err1
		}
	}

	delete(el.connections, c.fd)
	el.addConn(-1)
	if el.eventHandler.OnClose(c, err) == Shutdown {
		rerr = gerrors.ErrEngineShutdown
	}
	c.releaseTCP()

	return
}

func (el *eventloop) wake(c *conn) error {
	if co, ok := el.connections[c.fd]; !ok || co != c {
		return nil // ignore stale wakes.
	}

	action := el.eventHandler.OnTraffic(c)

	return el.handleAction(c, action)
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
		delay, action = el.eventHandler.OnTick()
		switch action {
		case None:
		case Shutdown:
			err := el.poller.UrgentTrigger(func(_ interface{}) error { return gerrors.ErrEngineShutdown }, nil)
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from OnTick(), UrgentTrigger:%v", el.idx, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			el.getLogger().Debugf("stopping ticker in event-loop(%d) from Engine, error:%v", el.idx, ctx.Err())
			return
		case <-timer.C:
		}
	}
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.closeConn(c, nil)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	default:
		return nil
	}
}

func (el *eventloop) readUDP(fd int, _ netpoll.IOEvent) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil
		}
		return fmt.Errorf("failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err))
	}
	var c *conn
	if fd == el.ln.fd {
		c = newUDPConn(fd, el, el.ln.addr, sa, false)
	} else {
		c = el.udpSockets[fd]
	}
	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)
	if c.peer != nil {
		c.releaseUDP()
	}
	if action == Shutdown {
		return gerrors.ErrEngineShutdown
	}
	return nil
}
