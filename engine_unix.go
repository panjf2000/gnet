// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package gnet

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/netpoll"
	"github.com/panjf2000/gnet/v2/pkg/queue"
	"github.com/panjf2000/gnet/v2/pkg/socket"
)

type engine struct {
	// Stop admission before joining event loops.
	admissionMu  sync.Mutex
	stopping     atomic.Bool
	enrollments  sync.WaitGroup
	listeners    map[int]*listener // listeners for accepting incoming connections
	opts         *Options          // options with engine
	ingress      *eventloop        // main event-loop that monitors all listeners
	eventLoops   loadBalancer      // event-loops for handling events
	inShutdown   atomic.Bool       // whether the engine is in shutdown
	turnOff      context.CancelFunc
	eventHandler EventHandler // user eventHandler
	concurrency  struct {
		*errgroup.Group

		ctx context.Context
	}
}

func (eng *engine) isShutdown() bool {
	return eng.inShutdown.Load()
}

func (eng *engine) isStopping() bool { return eng.stopping.Load() || eng.isShutdown() }

// shutdown signals the engine to shut down.
func (eng *engine) shutdown(err error) {
	eng.admissionMu.Lock()
	eng.stopping.Store(true)
	eng.admissionMu.Unlock()
	if err != nil && !errors.Is(err, errorx.ErrEngineShutdown) {
		eng.opts.Logger.Errorf("engine is being shutdown with error: %v", err)
	}
	// Cancel the context to stop the engine.
	eng.turnOff()
}

func (eng *engine) closeEventLoops() {
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		for _, ln := range el.listeners {
			ln.close()
		}
		_ = el.poller.Close()
		return true
	})
	if eng.ingress != nil {
		for _, ln := range eng.listeners {
			ln.close()
		}
		err := eng.ingress.poller.Close()
		if err != nil {
			eng.opts.Logger.Errorf("failed to close poller when stopping engine: %v", err)
		}
	}
}

func (eng *engine) runEventLoops(ctx context.Context, numEventLoop int) error {
	var el0 *eventloop
	lns := eng.listeners
	// Create loops locally and bind the listeners.
	for i := 0; i < numEventLoop; i++ {
		if i > 0 {
			lns = make(map[int]*listener, len(eng.listeners))
			for _, l := range eng.listeners {
				ln, err := initListener(l.network, l.address, eng.opts)
				if err != nil {
					for _, prepared := range lns {
						prepared.close()
					}
					return err
				}
				lns[ln.fd] = ln
			}
		}
		p, err := netpoll.OpenPoller()
		if err != nil {
			if i > 0 {
				for _, prepared := range lns {
					prepared.close()
				}
			}
			return err
		}
		el := new(eventloop)
		el.listeners = lns
		el.engine = eng
		el.poller = p
		el.buffer = make([]byte, eng.opts.ReadBufferCap)
		el.connections.init()
		el.eventHandler = eng.eventHandler
		// Transfer cleanup ownership before the first fallible poller update.
		eng.eventLoops.register(el)
		for _, ln := range lns {
			if err = el.poller.AddRead(ln.packPollAttachment(el.accept), false); err != nil {
				return err
			}
		}
		// Start the ticker.
		if eng.opts.Ticker && el.idx == 0 {
			el0 = el
		}
	}

	// Start event-loops in the background.
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.concurrency.Go(el.run)
		return true
	})

	if el0 != nil {
		eng.concurrency.Go(func() error {
			el0.ticker(ctx)
			return nil
		})
	}

	return nil
}

func (eng *engine) activateReactors(ctx context.Context, numEventLoop int) error {
	for i := 0; i < numEventLoop; i++ {
		p, err := netpoll.OpenPoller()
		if err != nil {
			return err
		}
		el := new(eventloop)
		el.listeners = eng.listeners
		el.engine = eng
		el.poller = p
		el.buffer = make([]byte, eng.opts.ReadBufferCap)
		el.connections.init()
		el.eventHandler = eng.eventHandler
		eng.eventLoops.register(el)
	}

	p, err := netpoll.OpenPoller()
	if err != nil {
		return err
	}
	el := new(eventloop)
	el.listeners = eng.listeners
	el.idx = -1
	el.engine = eng
	el.poller = p
	el.eventHandler = eng.eventHandler
	eng.ingress = el
	for _, ln := range eng.listeners {
		if err = el.poller.AddRead(ln.packPollAttachment(el.accept0), true); err != nil {
			return err
		}
	}
	// Start owners only after all pollers and listener registrations exist.
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.concurrency.Go(el.orbit)
		return true
	})

	// Start the main reactor in the background.
	eng.concurrency.Go(el.rotate)

	// Start the ticker.
	if eng.opts.Ticker {
		eng.concurrency.Go(func() error {
			eng.ingress.ticker(ctx)
			return nil
		})
	}

	return nil
}

func (eng *engine) start(ctx context.Context, numEventLoop int) error {
	if eng.opts.ReusePort {
		return eng.runEventLoops(ctx, numEventLoop)
	}

	return eng.activateReactors(ctx, numEventLoop)
}

func (eng *engine) stop(ctx context.Context, s Engine) {
	// Wait on a signal for shutdown
	<-ctx.Done()

	eng.eventHandler.OnShutdown(s)

	// Notify all event-loops to exit.
	eng.eventLoops.iterate(func(i int, el *eventloop) bool {
		err := el.poller.Trigger(queue.HighPriority, func(_ any) error { return errorx.ErrEngineShutdown }, nil)
		if err != nil {
			eng.opts.Logger.Errorf("failed to enqueue shutdown signal of high-priority for event-loop(%d): %v", i, err)
		}
		return true
	})
	if eng.ingress != nil {
		err := eng.ingress.poller.Trigger(queue.HighPriority, func(_ any) error { return errorx.ErrEngineShutdown }, nil)
		if err != nil {
			eng.opts.Logger.Errorf("failed to enqueue shutdown signal of high-priority for main event-loop: %v", err)
		}
	}

	if err := eng.concurrency.Wait(); err != nil {
		eng.opts.Logger.Errorf("engine shutdown error: %v", err)
	}

	eng.enrollments.Wait()
	// Close all listeners and pollers of event-loops.
	eng.closeEventLoops()

	// Put the engine into the shutdown state.
	eng.inShutdown.Store(true)
}

func run(eventHandler EventHandler, listeners []*listener, options *Options, addrs []string) error {
	numEventLoop := determineEventLoops(options)
	logging.Infof("Launching gnet with %d event-loops, listening on: %s",
		numEventLoop, strings.Join(addrs, " | "))

	lns := make(map[int]*listener, len(listeners))
	for _, ln := range listeners {
		lns[ln.fd] = ln
	}
	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)
	eng := engine{
		listeners:    lns,
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eventHandler,
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}
	switch options.LB {
	case RoundRobin:
		eng.eventLoops = new(roundRobinLoadBalancer)
	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	e := Engine{&eng}
	switch eng.eventHandler.OnBoot(e) {
	case None, Close:
	case Shutdown:
		return nil
	}

	if err := eng.start(ctx, numEventLoop); err != nil {
		eng.shutdown(err)
		eng.stop(rootCtx, e)
		eng.opts.Logger.Errorf("gnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(rootCtx, e)

	for _, addr := range addrs {
		allEngines.Store(addr, &eng)
	}

	return nil
}

func setKeepAlive(fd int, enabled bool, idle, intvl time.Duration, cnt int) error {
	if intvl == 0 {
		intvl = idle / 5
	}
	if cnt == 0 {
		cnt = 5
	}
	return socket.SetKeepAlive(fd, enabled, int(idle.Seconds()), int(intvl.Seconds()), cnt)
}

/*
func (eng *engine) sendCmd(cmd *asyncCmd, urgent bool) error {
	if !gfd.Validate(cmd.fd) {
		return errors.ErrInvalidConn
	}
	el := eng.eventLoops.index(cmd.fd.EventLoopIndex())
	if el == nil {
		return errors.ErrInvalidConn
	}
	if urgent {
		return el.poller.Trigger(queue.LowPriority, el.execCmd, cmd)
	}
	return el.poller.Trigger(el.execCmd, cmd)
}
*/
