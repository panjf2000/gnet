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
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

type engine struct {
	ln         *listener
	opts       *Options     // options with engine
	eventLoops loadBalancer // event-loops for handling events
	ticker     struct {
		ctx    context.Context
		cancel context.CancelFunc
	}
	inShutdown int32 // whether the engine is in shutdown
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
	if err != nil && err != errorx.ErrEngineShutdown {
		eng.opts.Logger.Errorf("engine is being shutdown with error: %v", err)
	}
	eng.workerPool.shutdown()
}

func (eng *engine) closeEventLoops() {
	eng.eventLoops.iterate(func(i int, el *eventloop) bool {
		el.ch <- errorx.ErrEngineShutdown
		return true
	})
	eng.ln.close()
}

func (eng *engine) start(numEventLoop int) error {
	for i := 0; i < numEventLoop; i++ {
		el := eventloop{
			ch:           make(chan interface{}, 1024),
			idx:          i,
			eng:          eng,
			connections:  make(map[*conn]struct{}),
			eventHandler: eng.eventHandler,
		}
		eng.eventLoops.register(&el)
		eng.workerPool.Go(el.run)
		if i == 0 && eng.opts.Ticker {
			eng.workerPool.Go(func() error {
				el.ticker(eng.ticker.ctx)
				return nil
			})
		}
	}

	eng.workerPool.Go(eng.listen)

	return nil
}

func (eng *engine) stop(engine Engine) error {
	<-eng.workerPool.shutdownCtx.Done()

	eng.opts.Logger.Infof("engine is being shutdown...")
	eng.eventHandler.OnShutdown(engine)

	if eng.ticker.cancel != nil {
		eng.ticker.cancel()
	}

	eng.closeEventLoops()

	if err := eng.workerPool.Wait(); err != nil {
		eng.opts.Logger.Errorf("engine shutdown error: %v", err)
	}

	atomic.StoreInt32(&eng.inShutdown, 1)

	return nil
}

func run(eventHandler EventHandler, listener *listener, options *Options, protoAddr string) error {
	// Figure out the proper number of event-loops/goroutines to run.
	numEventLoop := 1
	if options.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if options.NumEventLoop > 0 {
		numEventLoop = options.NumEventLoop
	}

	shutdownCtx, shutdown := context.WithCancel(context.Background())
	eng := engine{
		opts:         options,
		eventHandler: eventHandler,
		ln:           listener,
		workerPool: struct {
			*errgroup.Group
			shutdownCtx context.Context
			shutdown    context.CancelFunc
			once        sync.Once
		}{&errgroup.Group{}, shutdownCtx, shutdown, sync.Once{}},
	}

	switch options.LB {
	case RoundRobin:
		eng.eventLoops = new(roundRobinLoadBalancer)
	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	if options.Ticker {
		eng.ticker.ctx, eng.ticker.cancel = context.WithCancel(context.Background())
	}

	engine := Engine{eng: &eng}
	switch eventHandler.OnBoot(engine) {
	case None:
	case Shutdown:
		return nil
	}

	if err := eng.start(numEventLoop); err != nil {
		eng.opts.Logger.Errorf("gnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(engine) //nolint:errcheck

	allEngines.Store(protoAddr, &eng)

	return nil
}

/*
func (eng *engine) sendCmd(_ *asyncCmd, _ bool) error {
	return errorx.ErrUnsupportedOp
}
*/
