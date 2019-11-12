// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"time"

	"github.com/panjf2000/gnet/pool"
	"github.com/panjf2000/gnet/ringbuffer"
)

func (svr *server) listenerRun() {
	var err error
	defer svr.signalShutdown(err)
	var packet [0xFFFF]byte
	inBuf := svr.bytesPool.Get().(*ringbuffer.RingBuffer)
	bytesPool := pool.NewBytesPool()
	for {
		if svr.ln.pconn != nil {
			// Read data from UDP socket.
			n, addr, e := svr.ln.pconn.ReadFrom(packet[:])
			if e != nil {
				err = e
				return
			}
			buf := bytesPool.GetLen(n)
			copy(buf, packet[:n])

			lp := svr.subLoopGroup.next()
			c := &stdConn{
				loop:          lp,
				localAddr:     svr.ln.lnaddr,
				remoteAddr:    addr,
				inboundBuffer: inBuf,
				cache:         buf,
			}
			lp.ch <- &udpIn{c}
		} else {
			// Accept TCP socket.
			conn, e := svr.ln.ln.Accept()
			if e != nil {
				err = e
				return
			}
			lp := svr.subLoopGroup.next()
			c := newConn(conn, lp)
			lp.ch <- c
			go func() {
				var packet [0xFFFF]byte
				for {
					n, err := c.conn.Read(packet[:])
					if err != nil {
						_ = c.conn.SetReadDeadline(time.Time{})
						lp.ch <- &stderr{c, err}
						return
					}
					buf := bytesPool.GetLen(n)
					copy(buf, packet[:n])
					lp.ch <- &tcpIn{c, buf}
				}
			}()
		}
	}
}
