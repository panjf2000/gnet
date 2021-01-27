// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gnet

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	errors2 "github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/logging"
)

var errCloseAllConns = errors.New("close all connections in event-loop")

type server struct {
	ln           *listener          // the listeners for accepting new connections
	lb           loadBalancer       // event-loops for handling events
	cond         *sync.Cond         // shutdown signaler
	opts         *Options           // options with server
	serr         error              // signal error
	once         sync.Once          // make sure only signalShutdown once
	codec        ICodec             // codec for TCP stream
	loopWG       sync.WaitGroup     // loop close WaitGroup
	logger       logging.Logger     // customized logger for logging info
	ticktock     chan time.Duration // ticker channel
	listenerWG   sync.WaitGroup     // listener close WaitGroup
	inShutdown   int32              // whether the server is in shutdown
	eventHandler EventHandler       // user eventHandler
}

var serverFarm sync.Map

func (svr *server) isInShutdown() bool {
	return atomic.LoadInt32(&svr.inShutdown) == 1
}

// waitForShutdown waits for a signal to shutdown.
func (svr *server) waitForShutdown() error {
	svr.cond.L.Lock()
	svr.cond.Wait()
	err := svr.serr
	svr.cond.L.Unlock()
	return err
}

// signalShutdown signals the server to shut down.
func (svr *server) signalShutdown() {
	svr.signalShutdownWithErr(nil)
}

// signalShutdownWithErr signals the server to shut down with an error.
func (svr *server) signalShutdownWithErr(err error) {
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
		svr.listenerRun(svr.opts.LockOSThread)
		svr.listenerWG.Done()
	}()
}

func (svr *server) startEventLoops(numEventLoop int) {
	for i := 0; i < numEventLoop; i++ {
		el := new(eventloop)
		el.ch = make(chan interface{}, channelBuffer)
		el.svr = svr
		el.connections = make(map[*stdConn]struct{})
		el.eventHandler = svr.eventHandler
		el.calibrateCallback = svr.lb.calibrate
		svr.lb.register(el)
	}

	svr.loopWG.Add(svr.lb.len())
	svr.lb.iterate(func(i int, el *eventloop) bool {
		go el.loopRun(svr.opts.LockOSThread)
		return true
	})
}

func (svr *server) stop(s Server) {
	// Wait on a signal for shutdown.
	svr.logger.Infof("Server is being shutdown on the signal error: %v", svr.waitForShutdown())

	svr.eventHandler.OnShutdown(s)

	// Close listener.
	svr.ln.close()
	svr.listenerWG.Wait()

	// Notify all loops to close.
	svr.lb.iterate(func(i int, el *eventloop) bool {
		el.ch <- errors2.ErrServerShutdown
		return true
	})

	// Wait on all loops to close.
	svr.loopWG.Wait()

	// Close all connections.
	svr.loopWG.Add(svr.lb.len())
	svr.lb.iterate(func(i int, el *eventloop) bool {
		el.ch <- errCloseAllConns
		return true
	})
	svr.loopWG.Wait()

	atomic.StoreInt32(&svr.inShutdown, 1)
}

func serve(eventHandler EventHandler, listener *listener, options *Options, protoAddr string) (err error) {
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
		svr.lb = new(roundRobinLoadBalancer)
	case LeastConnections:
		svr.lb = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		svr.lb = new(sourceAddrHashLoadBalancer)
	}

	svr.ticktock = make(chan time.Duration, 1)
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.logger = logging.DefaultLogger
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
		return
	}

	// Start all event-loops in background.
	svr.startEventLoops(numEventLoop)

	// Start listener in background.
	svr.startListener()

	defer svr.stop(server)

	serverFarm.Store(protoAddr, svr)

	return
}
