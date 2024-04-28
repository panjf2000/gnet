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
// +build darwin dragonfly freebsd linux netbsd openbsd

package gnet

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/panjf2000/gnet/v2/internal/gfd"
	"github.com/panjf2000/gnet/v2/internal/netpoll"
	"github.com/panjf2000/gnet/v2/internal/queue"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type engine struct {
	listeners  map[int]*listener // listeners for accepting incoming connections
	opts       *Options          // options with engine
	ingress    *eventloop        // main event-loop that monitors all listeners
	eventLoops loadBalancer      // event-loops for handling events
	inShutdown int32             // whether the engine is in shutdown
	ticker     struct {
		ctx    context.Context    // context for ticker
		cancel context.CancelFunc // function to stop the ticker
	}
	workerPool struct {
		*errgroup.Group

		shutdownCtx context.Context
		shutdown    context.CancelFunc
		once        sync.Once
	}
	eventHandler EventHandler // user eventHandler
}

func (eng *engine) isInShutdown() bool {
	return atomic.LoadInt32(&eng.inShutdown) == 1
}

// shutdown signals the engine to shut down.
func (eng *engine) shutdown(err error) {
	if err != nil && err != errors.ErrEngineShutdown {
		eng.opts.Logger.Errorf("engine is being shutdown with error: %v", err)
	}

	eng.workerPool.once.Do(func() {
		eng.workerPool.shutdown()
	})
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

func (eng *engine) runEventLoops(numEventLoop int) error {
	var el0 *eventloop
	lns := eng.listeners
	// Create loops locally and bind the listeners.
	for i := 0; i < numEventLoop; i++ {
		if i > 0 {
			lns = make(map[int]*listener, len(eng.listeners))
			for _, l := range eng.listeners {
				ln, err := initListener(l.network, l.address, eng.opts)
				if err != nil {
					return err
				}
				lns[ln.fd] = ln
			}
		}
		p, err := netpoll.OpenPoller()
		if err != nil {
			return err
		}
		el := new(eventloop)
		el.listeners = lns
		el.engine = eng
		el.poller = p
		el.buffer = make([]byte, eng.opts.ReadBufferCap)
		el.connections.init()
		el.eventHandler = eng.eventHandler
		for _, ln := range lns {
			if err = el.poller.AddRead(ln.packPollAttachment(el.accept), false); err != nil {
				return err
			}
		}
		eng.eventLoops.register(el)

		// Start the ticker.
		if eng.opts.Ticker && el.idx == 0 {
			el0 = el
		}
	}

	// Start event-loops in background.
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.workerPool.Go(el.run)
		return true
	})

	if el0 != nil {
		eng.workerPool.Go(func() error {
			el0.ticker(eng.ticker.ctx)
			return nil
		})
	}

	return nil
}

func (eng *engine) activateReactors(numEventLoop int) error {
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

	// Start sub reactors in background.
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.workerPool.Go(el.orbit)
		return true
	})

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
	for _, ln := range eng.listeners {
		if err = el.poller.AddRead(ln.packPollAttachment(el.accept0), true); err != nil {
			return err
		}
	}
	eng.ingress = el

	// Start main reactor in background.
	eng.workerPool.Go(el.rotate)

	// Start the ticker.
	if eng.opts.Ticker {
		eng.workerPool.Go(func() error {
			eng.ingress.ticker(eng.ticker.ctx)
			return nil
		})
	}

	return nil
}

func (eng *engine) start(numEventLoop int) error {
	if eng.opts.ReusePort {
		return eng.runEventLoops(numEventLoop)
	}

	return eng.activateReactors(numEventLoop)
}

func (eng *engine) stop(s Engine) {
	// Wait on a signal for shutdown
	<-eng.workerPool.shutdownCtx.Done()

	eng.eventHandler.OnShutdown(s)

	// Notify all event-loops to exit.
	eng.eventLoops.iterate(func(i int, el *eventloop) bool {
		err := el.poller.Trigger(queue.HighPriority, func(_ interface{}) error { return errors.ErrEngineShutdown }, nil)
		if err != nil {
			eng.opts.Logger.Errorf("failed to enqueue shutdown signal of high-priority for event-loop(%d): %v", i, err)
		}
		return true
	})
	if eng.ingress != nil {
		err := eng.ingress.poller.Trigger(queue.HighPriority, func(_ interface{}) error { return errors.ErrEngineShutdown }, nil)
		if err != nil {
			eng.opts.Logger.Errorf("failed to enqueue shutdown signal of high-priority for main event-loop: %v", err)
		}
	}

	// Stop the ticker.
	if eng.ticker.cancel != nil {
		eng.ticker.cancel()
	}

	if err := eng.workerPool.Wait(); err != nil {
		eng.opts.Logger.Errorf("engine shutdown error: %v", err)
	}

	// Close all listeners and pollers of event-loops.
	eng.closeEventLoops()

	// Put the engine into the shutdown state.
	atomic.StoreInt32(&eng.inShutdown, 1)
}

func run(eventHandler EventHandler, listeners []*listener, options *Options, addrs []string) error {
	// Figure out the proper number of event-loop to run.
	numEventLoop := 1
	if options.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if options.NumEventLoop > 0 {
		numEventLoop = options.NumEventLoop
	}
	if numEventLoop > gfd.EventLoopIndexMax {
		numEventLoop = gfd.EventLoopIndexMax
	}

	logging.Infof("Launching gnet with %d event-loops, listening on: %s",
		numEventLoop, strings.Join(addrs, " | "))

	lns := make(map[int]*listener, len(listeners))
	for _, ln := range listeners {
		lns[ln.fd] = ln
	}
	shutdownCtx, shutdown := context.WithCancel(context.Background())
	eng := engine{
		listeners: lns,
		opts:      options,
		workerPool: struct {
			*errgroup.Group
			shutdownCtx context.Context
			shutdown    context.CancelFunc
			once        sync.Once
		}{&errgroup.Group{}, shutdownCtx, shutdown, sync.Once{}},
		eventHandler: eventHandler,
	}
	switch options.LB {
	case RoundRobin:
		eng.eventLoops = new(roundRobinLoadBalancer)
	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	if eng.opts.Ticker {
		eng.ticker.ctx, eng.ticker.cancel = context.WithCancel(context.Background())
	}

	e := Engine{&eng}
	switch eng.eventHandler.OnBoot(e) {
	case None:
	case Shutdown:
		return nil
	}

	if err := eng.start(numEventLoop); err != nil {
		eng.closeEventLoops()
		eng.opts.Logger.Errorf("gnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(e)

	for _, addr := range addrs {
		allEngines.Store(addr, &eng)
	}

	return nil
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
