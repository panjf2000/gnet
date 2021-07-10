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
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	gerrors "github.com/panjf2000/gnet/errors"
)

var errCloseAllConns = errors.New("close all connections in event-loop")

const TaskBufferCap = 256

type server struct {
	ln           *listener          // the listeners for accepting new connections
	lb           loadBalancer       // event-loops for handling events
	cond         *sync.Cond         // shutdown signaler
	opts         *Options           // options with server
	serr         error              // signal error
	once         sync.Once          // make sure only signalShutdown once
	codec        ICodec             // codec for TCP stream
	loopWG       sync.WaitGroup     // loop close WaitGroup
	listenerWG   sync.WaitGroup     // listener close WaitGroup
	inShutdown   int32              // whether the server is in shutdown
	tickerCtx    context.Context    // context for ticker
	cancelTicker context.CancelFunc // function to stop the ticker
	eventHandler EventHandler       // user eventHandler
}

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
	var striker *eventloop
	for i := 0; i < numEventLoop; i++ {
		el := new(eventloop)
		el.ch = make(chan interface{}, channelBuffer(TaskBufferCap))
		el.svr = svr
		el.connections = make(map[*stdConn]struct{})
		el.eventHandler = svr.eventHandler
		svr.lb.register(el)
		if el.idx == 0 && svr.opts.Ticker {
			striker = el
		}
	}

	svr.loopWG.Add(svr.lb.len())
	svr.lb.iterate(func(i int, el *eventloop) bool {
		go el.loopRun(svr.opts.LockOSThread)
		return true
	})

	// Start the ticker.
	go striker.loopTicker(svr.tickerCtx)
}

func (svr *server) stop(s Server) {
	// Wait on a signal for shutdown.
	svr.opts.Logger.Infof("Server is being shutdown on the signal error: %v", svr.waitForShutdown())

	svr.eventHandler.OnShutdown(s)

	// Close listener.
	svr.ln.close()
	svr.listenerWG.Wait()

	// Notify all loops to close.
	svr.lb.iterate(func(i int, el *eventloop) bool {
		el.ch <- gerrors.ErrServerShutdown
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

	// Stop the ticker.
	if svr.opts.Ticker {
		svr.cancelTicker()
	}

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

	if svr.opts.Ticker {
		svr.tickerCtx, svr.cancelTicker = context.WithCancel(context.Background())
	}
	svr.cond = sync.NewCond(&sync.Mutex{})
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

	allServers.Store(protoAddr, svr)

	return
}

// channelBuffer determines whether the channel should be a buffered channel to get the best performance.
func channelBuffer(n int) int {
	// Use blocking channel if GOMAXPROCS=1.
	// This switches context from sender to receiver immediately,
	// which results in higher performance.
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	// Make channel non-blocking and set up its capacity with GOMAXPROCS if GOMAXPROCS>1,
	// otherwise the sender might be dragged down if the receiver is CPU-bound.
	//
	// GOMAXPROCS determines how many goroutines can run in parallel,
	// which makes it the best choice as the channel capacity,
	return n
}
