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
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	errors2 "github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/logging"
)

// commandBufferSize represents the buffer size of event-loop command channel on Windows.
const (
	commandBufferSize = 512
)

var errCloseAllConns = errors.New("close all connections in event-loop")

type server struct {
	ln              *listener          // all the listeners
	cond            *sync.Cond         // shutdown signaler
	opts            *Options           // options with server
	serr            error              // signal error
	once            sync.Once          // make sure only signalShutdown once
	codec           ICodec             // codec for TCP stream
	loopWG          sync.WaitGroup     // loop close WaitGroup
	logger          logging.Logger     // customized logger for logging info
	ticktock        chan time.Duration // ticker channel
	listenerWG      sync.WaitGroup     // listener close WaitGroup
	eventHandler    EventHandler       // user eventHandler
	subEventLoopSet loadBalancer       // event-loops for handling events
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

func (svr *server) startEventLoops(numEventLoop int) {
	for i := 0; i < numEventLoop; i++ {
		el := &eventloop{
			ch:                make(chan interface{}, commandBufferSize),
			svr:               svr,
			codec:             svr.codec,
			connections:       make(map[*stdConn]struct{}),
			eventHandler:      svr.eventHandler,
			calibrateCallback: svr.subEventLoopSet.calibrate,
		}
		svr.subEventLoopSet.register(el)
	}

	svr.loopWG.Add(svr.subEventLoopSet.len())
	svr.subEventLoopSet.iterate(func(i int, el *eventloop) bool {
		go el.loopRun()
		return true
	})
}

func (svr *server) stop() {
	// Wait on a signal for shutdown.
	svr.logger.Infof("Server is being shutdown on the signal error: %v", svr.waitForShutdown())

	// Close listener.
	svr.ln.close()
	svr.listenerWG.Wait()

	// Notify all loops to close.
	svr.subEventLoopSet.iterate(func(i int, el *eventloop) bool {
		el.ch <- errors2.ErrServerShutdown
		return true
	})

	// Wait on all loops to close.
	svr.loopWG.Wait()

	// Close all connections.
	svr.loopWG.Add(svr.subEventLoopSet.len())
	svr.subEventLoopSet.iterate(func(i int, el *eventloop) bool {
		el.ch <- errCloseAllConns
		return true
	})
	svr.loopWG.Wait()
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

	switch options.LB {
	case RoundRobin:
		svr.subEventLoopSet = new(roundRobinEventLoopSet)
	case LeastConnections:
		svr.subEventLoopSet = new(leastConnectionsEventLoopSet)
	case SourceAddrHash:
		svr.subEventLoopSet = new(sourceAddrHashEventLoopSet)
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
	defer svr.eventHandler.OnShutdown(server)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer close(shutdown)

	go func() {
		if <-shutdown == nil {
			return
		}
		svr.signalShutdown(errors.New("caught OS signal"))
	}()

	// Start all event-loops in background.
	svr.startEventLoops(numEventLoop)

	// Start listener in background.
	svr.startListener()

	defer svr.stop()

	return
}
