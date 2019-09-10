package gnet

import (
	"sync/atomic"
	"time"

	"github.com/smartystreets-prototypes/go-disruptor"
	"golang.org/x/sys/unix"
)

const (
	RingBufferSize   = 1024
	RingBufferMask   = RingBufferSize - 1
	ReserveMany      = 16
	ReserveManyDelta = ReserveMany - 1
	DisruptorCleanup = time.Millisecond * 10
)

var ringBuffer = [RingBufferSize]*conn{}

type eventConsumer struct {
	s *server
	l *loop
}

func (ec *eventConsumer) Consume(lower, upper int64) {
	for lower <= upper {
		c := ringBuffer[lower&RingBufferMask]
		ec.l.fdconns[c.fd] = c
		c.loop = ec.l
		if len(ec.s.loops) > 1 {
			switch ec.s.balance {
			case RoundRobin:
				idx := int(atomic.LoadUintptr(&ec.s.accepted)) % len(ec.s.loops)
				if idx != ec.l.idx {
					return // do not accept
				}
				atomic.AddUintptr(&ec.s.accepted, 1)
			case LeastConnections:
				n := atomic.LoadInt32(&ec.l.count)
				for _, lp := range ec.s.loops {
					if lp.idx != ec.l.idx {
						if atomic.LoadInt32(&lp.count) < n {
							return // do not accept
						}
					}
				}
				atomic.AddInt32(&ec.l.count, 1)
			}
		}
		ec.l.poll.AddReadWrite(c.fd)
		lower++
	}
}

func activateMainReactor(s *server, l *loop) {
	defer func() {
		s.signalShutdown()
		s.wg.Done()
	}()

	if s.events.Tick != nil {
		go loopTicker(s, l)
	}

	eventConsumers := make([]disruptor.Consumer, 0, len(s.loops))
	for _, l := range s.loops {
		ec := &eventConsumer{s, l}
		eventConsumers = append(eventConsumers, ec)
	}

	// Initialize go-disruptor with ring-buffer for dispatching events to loops.
	controller := disruptor.Configure(RingBufferSize).WithConsumerGroup(eventConsumers...).Build()
	controller.Start()
	defer controller.Stop()
	writer := controller.Writer()
	sequence := disruptor.InitialSequenceValue

	_ = l.poll.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return loopNote(s, l, note)
		}

		for i, ln := range s.lns {
			if ln.fd == fd {
				if ln.pconn != nil {
					return loopUDPRead(s, l, i, fd)
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
				c := &conn{fd: nfd, sa: sa, lnidx: i}
				sequence = writer.Reserve(1)
				ringBuffer[sequence&RingBufferMask] = c
				writer.Commit(sequence, sequence)
				return nil
			}
		}
		return nil
	})
}

func activateSubReactor(s *server, l *loop) {
	defer func() {
		s.signalShutdown()
		s.wg.Done()
	}()

	_ = l.poll.Polling(func(fd int, note interface{}) error {
		if fd == 0 {
			return loopNote(s, l, note)
		}

		c := l.fdconns[fd]
		switch {
		case !c.opened:
			return loopOpened(s, l, c)
		case len(c.out) > 0:
			return loopWrite(s, l, c)
		case c.action != None:
			return loopAction(s, l, c)
		default:
			return loopRead(s, l, c)
		}
	})
}
