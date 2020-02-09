// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux darwin netbsd freebsd openbsd dragonfly

package gnet

import (
	"log"
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

func (svr *server) activateLoops(numLoops int) error {
	// Create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			el := &eventloop{
				idx:          i,
				svr:          svr,
				codec:        svr.codec,
				poller:       p,
				packet:       make([]byte, 0xFFFF),
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

func (svr *server) activateReactors(numLoops int) error {
	for i := 0; i < numLoops; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			el := &eventloop{
				idx:          i,
				svr:          svr,
				codec:        svr.codec,
				poller:       p,
				packet:       make([]byte, 0xFFFF),
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

func (svr *server) start(numCPU int) error {
	if svr.opts.ReusePort || svr.ln.pconn != nil {
		return svr.activateLoops(numCPU)
	}
	return svr.activateReactors(numCPU)
}

func (svr *server) stop() {
	// Wait on a signal for shutdown
	svr.waitForShutdown()

	// Notify all loops to close by closing all listeners
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		sniffError(el.poller.Trigger(func() error {
			return errServerShutdown
		}))
		return true
	})

	if svr.mainLoop != nil {
		svr.ln.close()
		sniffError(svr.mainLoop.poller.Trigger(func() error {
			return errServerShutdown
		}))
	}

	// Wait on all loops to complete reading events
	svr.wg.Wait()

	// Close loops and all outstanding connections
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		for _, c := range el.connections {
			sniffError(el.loopCloseConn(c, nil))
		}
		return true
	})
	svr.closeLoops()

	if svr.mainLoop != nil {
		sniffError(svr.mainLoop.poller.Close())
	}
}

func serve(eventHandler EventHandler, listener *listener, options *Options) error {
	// Figure out the correct number of loops/goroutines to use.
	var numCPU int
	if options.Multicore {
		numCPU = runtime.NumCPU()
	} else {
		numCPU = 1
	}

	svr := new(server)
	svr.opts = options
	svr.eventHandler = eventHandler
	svr.ln = listener
	svr.subLoopGroup = new(eventLoopGroup)
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.ticktock = make(chan time.Duration, 1)
	svr.codec = func() ICodec {
		if options.Codec == nil {
			return new(BuiltInFrameCodec)
		}
		return options.Codec
	}()

	server := Server{
		Multicore:    options.Multicore,
		Addr:         listener.lnaddr,
		NumLoops:     numCPU,
		ReUsePort:    options.ReusePort,
		TCPKeepAlive: options.TCPKeepAlive,
	}
	switch svr.eventHandler.OnInitComplete(server) {
	case None:
	case Shutdown:
		return nil
	}

	if err := svr.start(numCPU); err != nil {
		svr.closeLoops()
		log.Printf("gnet server is stoping with error: %v\n", err)
		return err
	}
	defer svr.stop()

	return nil
}
