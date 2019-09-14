// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"time"

	"github.com/panjf2000/gnet/ringbuffer"
	"github.com/smartystreets-prototypes/go-disruptor"
	"golang.org/x/sys/unix"
)

const (
	RingBufferSize   = 1024
	RingBufferMask   = RingBufferSize - 1
	DisruptorCleanup = time.Millisecond * 10
)

type mail struct {
	fd   int
	conn *conn
}

type eventConsumer struct {
	numLoops       int
	loop           *loop
	connRingBuffer *[RingBufferSize]*conn
}

func (ec *eventConsumer) Consume(lower, upper int64) {
	//fmt.Printf("consumer with loop: %d, consuming message, lower: %d, upper: %d\n", ec.loop.idx, lower, upper)
	for ; lower <= upper; lower++ {
		conn := ec.connRingBuffer[lower&RingBufferMask]
		conn.inBuf = ringbuffer.New(RingBufferSize)
		conn.outBuf = ringbuffer.New(RingBufferSize)
		//fmt.Printf("lower: %d, consuming fd: %d in loop: %d\n", lower, conn.fd, ec.loop.idx)
		conn.loop = ec.loop

		// Connections load balance under round-robin algorithm.
		if ec.numLoops > 1 {
			idx := int(lower) % ec.numLoops
			if idx != ec.loop.idx {
				//fmt.Printf("lower: %d, ignoring fd: %d in loop: %d\n", lower, conn.fd, ec.loop.idx)
				// Don't match the round-robin rule, ignore this connection.
				continue
			}
		}
		//fmt.Printf("lower: %d, sendOut fd: %d to loop: %d\n", lower, conn.fd, ec.loop.idx)
		_ = ec.loop.poller.Trigger(&mail{fd: conn.fd, conn: conn})
	}
}

func activateMainReactor(svr *server) {
	defer func() {
		time.Sleep(DisruptorCleanup)
		svr.signalShutdown()
		svr.wg.Done()
	}()

	var connRingBuffer = &[RingBufferSize]*conn{}

	eventConsumers := make([]disruptor.Consumer, 0, svr.numLoops)
	for _, loop := range svr.loops {
		ec := &eventConsumer{svr.numLoops, loop, connRingBuffer}
		eventConsumers = append(eventConsumers, ec)
	}
	//fmt.Printf("length of loops: %d and consumers: %d\n", svr.numLoops, len(eventConsumers))

	// Initialize go-disruptor with ring-buffer for dispatching events to loops.
	controller := disruptor.Configure(RingBufferSize).WithConsumerGroup(eventConsumers...).Build()

	controller.Start()
	defer controller.Stop()

	writer := controller.Writer()
	sequence := disruptor.InitialSequenceValue

	//fmt.Println("main reactor polling...")
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
				//fmt.Printf("accepted fd: %d in main reactor\n", nfd)
				sequence = writer.Reserve(1)
				connRingBuffer[sequence&RingBufferMask] = conn
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
		//fmt.Println("start ticker...")
		go loop.loopTicker(svr)
	}

	//fmt.Printf("sub reactor polling, loop: %d\n", loop.idx)
	_ = loop.poller.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return loop.loopNote(svr, note)
		}

		conn := loop.fdconns[fd]
		switch {
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
