// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"runtime"
	"sync"
	"time"

	"github.com/panjf2000/gnet/internal/netpoll"
)

type server struct {
	ln               *listener          // all the listeners
	wg               sync.WaitGroup     // event-loop close WaitGroup
	opts             *Options           // options with server
	once             sync.Once          // make sure only signalShutdown once
	cond             *sync.Cond         // shutdown signaler
	codec            ICodec             // codec for TCP stream
	logger           Logger             // customized logger for logging info
	ticktock         chan time.Duration // ticker channel
	mainLoop         *eventloop         // main loop for accepting connections
	eventHandler     EventHandler       // user eventHandler
	subLoopGroup     IEventLoopGroup    // loops for handling events
	subLoopGroupSize int                // number of loops
}

// waitForShutdown waits for a signal to shutdown
func (svr *server) waitForShutdown() {
	svr.cond.L.Lock()
	svr.cond.Wait()
	svr.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins server closing
func (svr *server) signalShutdown() {
	svr.once.Do(func() {
		svr.cond.L.Lock()
		svr.cond.Signal()
		svr.cond.L.Unlock()
	})
}

func (svr *server) startLoops() {
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		svr.wg.Add(1)
		go func() {
			el.loopRun()
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) closeLoops() {
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		_ = el.poller.Close()
		return true
	})
}

func (svr *server) startReactors() {
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		svr.wg.Add(1)
		go func() {
			svr.activateSubReactor(el)
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) activateLoops(numEventLoop int) error {
	// Create loops locally and bind the listeners.
	for i := 0; i < numEventLoop; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			el := &eventloop{
				idx:          i,
				svr:          svr,
				codec:        svr.codec,
				poller:       p,
				packet:       make([]byte, 0x10000),
				connections:  make(map[int]*conn),
				eventHandler: svr.eventHandler,
			}
			_ = el.poller.AddRead(svr.ln.fd)
			svr.subLoopGroup.register(el)
		} else {
			return err
		}
	}
	svr.subLoopGroupSize = svr.subLoopGroup.len()
	// Start loops in background
	svr.startLoops()
	return nil
}

func (svr *server) activateReactors(numEventLoop int) error {
	for i := 0; i < numEventLoop; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			el := &eventloop{
				idx:          i,
				svr:          svr,
				codec:        svr.codec,
				poller:       p,
				packet:       make([]byte, 0x10000),
				connections:  make(map[int]*conn),
				eventHandler: svr.eventHandler,
			}
			svr.subLoopGroup.register(el)
		} else {
			return err
		}
	}
	svr.subLoopGroupSize = svr.subLoopGroup.len()
	// Start sub reactors.
	svr.startReactors()

	if p, err := netpoll.OpenPoller(); err == nil {
		el := &eventloop{
			idx:    -1,
			poller: p,
			svr:    svr,
		}
		_ = el.poller.AddRead(svr.ln.fd)
		svr.mainLoop = el
		// Start main reactor.
		svr.wg.Add(1)
		go func() {
			svr.activateMainReactor()
			svr.wg.Done()
		}()
	} else {
		return err
	}
	return nil
}

func (svr *server) start(numEventLoop int) error {
	if svr.opts.ReusePort || svr.ln.pconn != nil {
		return svr.activateLoops(numEventLoop)
	}
	return svr.activateReactors(numEventLoop)
}

func (svr *server) stop() {
	// Wait on a signal for shutdown
	svr.waitForShutdown()

	// Notify all loops to close by closing all listeners
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		sniffErrorAndLog(el.poller.Trigger(func() error {
			return ErrServerShutdown
		}))
		return true
	})

	if svr.mainLoop != nil {
		svr.ln.close()
		sniffErrorAndLog(svr.mainLoop.poller.Trigger(func() error {
			return ErrServerShutdown
		}))
	}

	// Wait on all loops to complete reading events
	svr.wg.Wait()

	// Close loops and all outstanding connections
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		for _, c := range el.connections {
			sniffErrorAndLog(el.loopCloseConn(c, nil))
		}
		return true
	})
	svr.closeLoops()

	if svr.mainLoop != nil {
		sniffErrorAndLog(svr.mainLoop.poller.Close())
	}
}

func serve(eventHandler EventHandler, listener *listener, options *Options) error {
	// Figure out the correct number of loops/goroutines to use.
	numEventLoop := 1
	if options.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if options.NumEventLoop > 0 {
		numEventLoop = options.NumEventLoop
	}

	svr := new(server)
	svr.opts = options
	svr.eventHandler = eventHandler
	svr.ln = listener

	switch options.LB {
	case RoundRobin:
		svr.subLoopGroup = new(roundRobinEventLoopGroup)
	case LeastConnections:
		svr.subLoopGroup = new(leastConnectionsEventLoopGroup)
	case SourceAddrHash:
		svr.subLoopGroup = new(sourceAddrHashEventLoopGroup)
	}

	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.ticktock = make(chan time.Duration, 1)
	svr.logger = func() Logger {
		if options.Logger == nil {
			return defaultLogger
		}
		return options.Logger
	}()
	svr.codec = func() ICodec {
		if options.Codec == nil {
			return new(BuiltInFrameCodec)
		}
		return options.Codec
	}()

	server := Server{
		svr:          svr,
		Multicore:    options.Multicore,
		Addr:         listener.lnaddr,
		NumEventLoop: numEventLoop,
		ReusePort:    options.ReusePort,
		TCPKeepAlive: options.TCPKeepAlive,
	}
	switch svr.eventHandler.OnInitComplete(server) {
	case None:
	case Shutdown:
		return nil
	}

	if err := svr.start(numEventLoop); err != nil {
		svr.closeLoops()
		svr.logger.Printf("gnet server is stoping with error: %v\n", err)
		return err
	}
	defer svr.stop()

	return nil
}
