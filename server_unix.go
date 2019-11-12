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

	"github.com/panjf2000/gnet/netpoll"
	"github.com/panjf2000/gnet/ringbuffer"
)

type server struct {
	ln               *listener          // all the listeners
	wg               sync.WaitGroup     // loop close WaitGroup
	tch              chan time.Duration // ticker channel
	opts             *Options           // options with server
	once             sync.Once          // make sure only signalShutdown once
	cond             *sync.Cond         // shutdown signaler
	codec            ICodec             // codec for TCP stream
	mainLoop         *loop              // main loop for accepting connections
	bytesPool        sync.Pool          // pool for storing bytes
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
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		svr.wg.Add(1)
		go func() {
			lp.loopRun()
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) closeLoops() {
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		_ = lp.poller.Close()
		return true
	})
}

func (svr *server) startReactors() {
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		svr.wg.Add(1)
		go func() {
			svr.activateSubReactor(lp)
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) activateLoops(numLoops int) error {
	// Create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			lp := &loop{
				idx:         i,
				poller:      p,
				packet:      make([]byte, 0xFFFF),
				connections: make(map[int]*conn),
				svr:         svr,
			}
			_ = lp.poller.AddRead(svr.ln.fd)
			svr.subLoopGroup.register(lp)
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
			lp := &loop{
				idx:         i,
				poller:      p,
				packet:      make([]byte, 0xFFFF),
				connections: make(map[int]*conn),
				svr:         svr,
			}
			svr.subLoopGroup.register(lp)
		} else {
			return err
		}
	}
	svr.subLoopGroupSize = svr.subLoopGroup.len()
	// Start sub reactors.
	svr.startReactors()

	if p, err := netpoll.OpenPoller(); err == nil {
		lp := &loop{
			idx:    -1,
			poller: p,
			svr:    svr,
		}
		_ = lp.poller.AddRead(svr.ln.fd)
		svr.mainLoop = lp
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
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		sniffError(lp.poller.Trigger(func() error {
			return errShutdown
		}))
		return true
	})

	if svr.mainLoop != nil {
		sniffError(svr.mainLoop.poller.Trigger(func() error {
			return errShutdown
		}))
	}

	// Wait on all loops to complete reading events
	svr.wg.Wait()

	// Close loops and all outstanding connections
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		for _, c := range lp.connections {
			sniffError(lp.loopCloseConn(c, nil))
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
	svr.eventHandler = eventHandler
	svr.ln = listener
	svr.subLoopGroup = new(eventLoopGroup)
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.tch = make(chan time.Duration)
	svr.opts = options
	svr.bytesPool.New = func() interface{} {
		return ringbuffer.New(socketRingBufferSize)
	}
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
