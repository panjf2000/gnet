// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package gnet

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

// commandBufferSize represents the buffer size of event-loop command channel on Windows.
const (
	commandBufferSize = 512
)

var (
	errClosing    = errors.New("closing")
	errCloseConns = errors.New("close conns")
)

type server struct {
	ln               *listener          // all the listeners
	cond             *sync.Cond         // shutdown signaler
	opts             *Options           // options with server
	serr             error              // signal error
	once             sync.Once          // make sure only signalShutdown once
	codec            ICodec             // codec for TCP stream
	loops            []*eventloop       // all the loops
	loopWG           sync.WaitGroup     // loop close WaitGroup
	logger           Logger             // customized logger for logging info
	ticktock         chan time.Duration // ticker channel
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

func (svr *server) startLoops(numEventLoop int) {
	for i := 0; i < numEventLoop; i++ {
		el := &eventloop{
			ch:           make(chan interface{}, commandBufferSize),
			idx:          i,
			svr:          svr,
			codec:        svr.codec,
			connections:  make(map[*stdConn]bool),
			eventHandler: svr.eventHandler,
		}
		svr.subLoopGroup.register(el)
	}
	svr.subLoopGroupSize = svr.subLoopGroup.len()
	svr.loopWG.Add(svr.subLoopGroupSize)
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		go el.loopRun()
		return true
	})
}

func (svr *server) stop() {
	// Wait on a signal for shutdown.
	svr.logger.Printf("server is being shutdown with err: %v\n", svr.waitForShutdown())

	// Close listener.
	svr.ln.close()
	svr.listenerWG.Wait()

	// Notify all loops to close.
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		el.ch <- errClosing
		return true
	})

	// Wait on all loops to close.
	svr.loopWG.Wait()

	// Close all connections.
	svr.loopWG.Add(svr.subLoopGroupSize)
	svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		el.ch <- errCloseConns
		return true
	})
	svr.loopWG.Wait()
	return
}

func serve(eventHandler EventHandler, listener *listener, options *Options) (err error) {
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
	svr.subLoopGroup = new(eventLoopGroup)
	svr.ticktock = make(chan time.Duration, 1)
	svr.cond = sync.NewCond(&sync.Mutex{})
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
		Multicore:    options.Multicore,
		Addr:         listener.lnaddr,
		NumEventLoop: numEventLoop,
		ReusePort:    options.ReusePort,
		TCPKeepAlive: options.TCPKeepAlive,
	}
	switch svr.eventHandler.OnInitComplete(server) {
	case None:
	case Shutdown:
		return
	}

	// Start all loops.
	svr.startLoops(numEventLoop)
	// Start listener.
	svr.startListener()
	defer svr.stop()

	return
}
