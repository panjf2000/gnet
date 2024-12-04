// Copyright (c) 2023 The Gnet Authors. All rights reserved.
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

package gnet

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type engine struct {
	listeners     []*listener
	opts          *Options     // options with engine
	eventLoops    loadBalancer // event-loops for handling events
	inShutdown    atomic.Bool  // whether the engine is in shutdown
	beingShutdown atomic.Bool  // whether the engine is being shutdown
	turnOff       context.CancelFunc
	eventHandler  EventHandler // user eventHandler
	concurrency   struct {
		*errgroup.Group

		ctx context.Context
	}
}

func (eng *engine) isShutdown() bool {
	return eng.inShutdown.Load()
}

// shutdown signals the engine to shut down.
func (eng *engine) shutdown(err error) {
	if err != nil && !errors.Is(err, errorx.ErrEngineShutdown) {
		eng.opts.Logger.Errorf("engine is being shutdown with error: %v", err)
	}
	eng.turnOff()
	eng.beingShutdown.Store(true)
}

func (eng *engine) closeEventLoops() {
	eng.eventLoops.iterate(func(i int, el *eventloop) bool {
		el.ch <- errorx.ErrEngineShutdown
		return true
	})
	for _, ln := range eng.listeners {
		ln.close()
	}
}

func (eng *engine) start(ctx context.Context, numEventLoop int) error {
	for i := 0; i < numEventLoop; i++ {
		el := eventloop{
			ch:           make(chan any, 1024),
			idx:          i,
			eng:          eng,
			connections:  make(map[*conn]struct{}),
			eventHandler: eng.eventHandler,
		}
		eng.eventLoops.register(&el)
		eng.concurrency.Go(el.run)
		if i == 0 && eng.opts.Ticker {
			eng.concurrency.Go(func() error {
				el.ticker(ctx)
				return nil
			})
		}
	}

	for _, ln := range eng.listeners {
		l := ln
		if l.pc != nil {
			eng.concurrency.Go(func() error {
				return eng.ListenUDP(l.pc)
			})
		} else {
			eng.concurrency.Go(func() error {
				return eng.listenStream(l.ln)
			})
		}
	}

	return nil
}

func (eng *engine) stop(ctx context.Context, engine Engine) {
	<-ctx.Done()

	eng.eventHandler.OnShutdown(engine)

	eng.closeEventLoops()

	if err := eng.concurrency.Wait(); err != nil && !errors.Is(err, errorx.ErrEngineShutdown) {
		eng.opts.Logger.Errorf("engine shutdown error: %v", err)
	}

	eng.inShutdown.Store(true)
}

func run(eventHandler EventHandler, listeners []*listener, options *Options, addrs []string) error {
	// Figure out the proper number of event-loops/goroutines to run.
	numEventLoop := 1
	if options.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if options.NumEventLoop > 0 {
		numEventLoop = options.NumEventLoop
	}

	logging.Infof("Launching gnet with %d event-loops, listening on: %s",
		numEventLoop, strings.Join(addrs, " | "))

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)
	eng := engine{
		opts:         options,
		listeners:    listeners,
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
		// If there are more than one listener, we can't use roundRobinLoadBalancer because
		// it's not concurrency-safe, replace it with leastConnectionsLoadBalancer.
		if len(listeners) > 1 {
			eng.eventLoops = new(leastConnectionsLoadBalancer)
		}
	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	engine := Engine{eng: &eng}
	switch eventHandler.OnBoot(engine) {
	case None, Close:
	case Shutdown:
		return nil
	}

	if err := eng.start(ctx, numEventLoop); err != nil {
		eng.opts.Logger.Errorf("gnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(rootCtx, engine)

	for _, addr := range addrs {
		allEngines.Store(addr, &eng)
	}

	return nil
}

/*
func (eng *engine) sendCmd(_ *asyncCmd, _ bool) error {
	return errorx.ErrUnsupportedOp
}
*/
