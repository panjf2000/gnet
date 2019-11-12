// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"errors"
	"github.com/panjf2000/gnet/ringbuffer"
	"log"
	"runtime"
	"sync"
	"time"
)

var errClosing = errors.New("closing")
var errCloseConns = errors.New("close conns")

type server struct {
	ln               *listener          // all the listeners
	cond             *sync.Cond         // shutdown signaler
	opts             *Options           // options with server
	serr             error              // signal error
	once             sync.Once          // make sure only signalShutdown once
	codec            ICodec             // codec for TCP stream
	loops            []*loop            // all the loops
	loopWG           sync.WaitGroup     // loop close WaitGroup
	ticktock         chan time.Duration // ticker channel
	bytesPool        sync.Pool          // pool for storing bytes
	listenerWG       sync.WaitGroup     // listener close WaitGroup
	eventHandler     EventHandler       // user eventHandler
	subLoopGroup     IEventLoopGroup    // loops for handling events
	subLoopGroupSize int                // number of loops
}

// waitForShutdown waits for a signal to shutdown.
func (svr *server) waitForShutdown() error {
	svr.cond.L.Lock()
	svr.cond.Wait()
	err := svr.serr
	svr.cond.L.Unlock()
	return err
}

// signalShutdown signals a shutdown an begins server closing.
func (svr *server) signalShutdown(err error) {
	svr.once.Do(func() {
		svr.cond.L.Lock()
		svr.serr = err
		svr.cond.Signal()
		svr.cond.L.Unlock()
	})
}

func (svr *server) startListener() {
	svr.listenerWG.Add(1)
	go func() {
		svr.listenerRun()
		svr.listenerWG.Done()
	}()
}

func (svr *server) startLoops(numLoops int) {
	for i := 0; i < numLoops; i++ {
		lp := &loop{
			idx:         i,
			ch:          make(chan interface{}, 64),
			connections: make(map[*stdConn]bool),
			svr:         svr,
		}
		svr.subLoopGroup.register(lp)
	}
	svr.subLoopGroupSize = svr.subLoopGroup.len()
	svr.loopWG.Add(svr.subLoopGroupSize)
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		go lp.loopRun()
		return true
	})
}

func (svr *server) stop() {
	// Wait on a signal for shutdown.
	log.Printf("server is being shutdown with err: %v\n", svr.waitForShutdown())

	// Close listener.
	svr.ln.close()
	svr.listenerWG.Wait()

	// Notify all loops to close.
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		lp.ch <- errClosing
		return true
	})

	// Wait on all loops to close.
	svr.loopWG.Wait()

	// Close all connections.
	svr.loopWG.Add(svr.subLoopGroupSize)
	svr.subLoopGroup.iterate(func(i int, lp *loop) bool {
		lp.ch <- errCloseConns
		return true
	})
	svr.loopWG.Wait()
	return
}

func serve(eventHandler EventHandler, listener *listener, options *Options) (err error) {
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
	svr.ticktock = make(chan time.Duration, 1)
	svr.cond = sync.NewCond(&sync.Mutex{})
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
		return
	}

	// Start all loops.
	svr.startLoops(numCPU)
	// Start listener.
	svr.startListener()
	defer svr.stop()

	return
}
