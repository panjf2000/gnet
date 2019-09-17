// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"time"

	"github.com/panjf2000/gnet/ringbuffer"
	"github.com/smartystreets-prototypes/go-disruptor"
	"golang.org/x/sys/unix"
)

const (
	// connRingBufferSize indicates the size of disruptor ring-buffer.
	connRingBufferSize = 1024 * 64
	connRingBufferMask = connRingBufferSize - 1
	disruptorCleanup   = time.Millisecond * 10

	cacheRingBufferSize = 1024
)

type mail struct {
	fd   int
	conn *conn
}

type eventConsumer struct {
	numLoops       int
	numLoopsMask   int
	loop           *loop
	connRingBuffer *[connRingBufferSize]*conn
}

func (ec *eventConsumer) Consume(lower, upper int64) {
	for ; lower <= upper; lower++ {
		// Connections load balance under round-robin algorithm.
		if ec.numLoops > 1 {
			// Leverage "and" operator instead of "modulo" operator to speed up round-robin algorithm.
			idx := int(lower) & ec.numLoopsMask
			if idx != ec.loop.idx {
				// Don't match the round-robin rule, ignore this connection.
				continue
			}
		}

		conn := ec.connRingBuffer[lower&connRingBufferMask]
		conn.inBuf = ringbuffer.New(cacheRingBufferSize)
		conn.outBuf = ringbuffer.New(cacheRingBufferSize)
		conn.loop = ec.loop

		_ = ec.loop.poller.Trigger(&mail{fd: conn.fd, conn: conn})
	}
}

func activateMainReactor(svr *server) {
	defer func() {
		time.Sleep(disruptorCleanup)
		svr.signalShutdown()
		svr.wg.Done()
	}()

	var connRingBuffer = &[connRingBufferSize]*conn{}

	eventConsumers := make([]disruptor.Consumer, 0, svr.numLoops)
	for _, loop := range svr.loops {
		ec := &eventConsumer{svr.numLoops, svr.numLoops - 1, loop, connRingBuffer}
		eventConsumers = append(eventConsumers, ec)
	}

	// Initialize go-disruptor with ring-buffer for dispatching events to loops.
	controller := disruptor.Configure(connRingBufferSize).WithConsumerGroup(eventConsumers...).Build()

	controller.Start()
	defer controller.Stop()

	writer := controller.Writer()
	sequence := disruptor.InitialSequenceValue

	_ = svr.mainLoop.poller.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return svr.mainLoop.loopNote(svr, note)
		}

		for i, ln := range svr.lns {
			if ln.fd == fd {
				if ln.pconn != nil {
					return svr.mainLoop.loopUDPRead(svr, i, fd)
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
				conn := &conn{fd: nfd, sa: sa, lnidx: i}
				sequence = writer.Reserve(1)
				connRingBuffer[sequence&connRingBufferMask] = conn
				writer.Commit(sequence, sequence)
				return nil
			}
		}
		return nil
	})
}

func activateSubReactor(svr *server, loop *loop) {
	defer func() {
		svr.signalShutdown()
		svr.wg.Done()
	}()

	if loop.idx == 0 && svr.events.Tick != nil {
		go loop.loopTicker(svr)
	}

	_ = loop.poller.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return loop.loopNote(svr, note)
		}

		conn := loop.fdconns[fd]
		switch {
		case conn == nil:
			return nil
		case !conn.opened:
			return loop.loopOpened(svr, conn)
		case conn.outBuf.Length() > 0:
			return loop.loopWrite(svr, conn)
		case conn.action != None:
			return loop.loopAction(svr, conn)
		default:
			return loop.loopRead(svr, conn)
		}
	})
}
