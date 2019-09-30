// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/panjf2000/gnet/netpoll"
	"golang.org/x/sys/unix"
)

type server struct {
	eventHandler   EventHandler // user eventHandler
	mainLoop       *loop
	eventLoopGroup eventLoopGrouper
	loopGroupSize  int // number of loops
	opts           *Options
	ln             *listener          // all the listeners
	wg             sync.WaitGroup     // loop close waitgroup
	cond           *sync.Cond         // shutdown signaler
	tch            chan time.Duration // ticker channel
	once           sync.Once          // make sure only signalShutdown once
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
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		svr.wg.Add(1)
		go func() {
			l.loopRun()
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) closeLoops() {
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		_ = l.poller.Close()
		return true
	})
}

func (svr *server) startReactors() {
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		svr.wg.Add(1)
		go func() {
			svr.activateSubReactor(l)
			svr.wg.Done()
		}()
		return true
	})
}

func (svr *server) activateLoops(numLoops int) error {
	// Create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			loop := &loop{
				idx:         i,
				poller:      p,
				packet:      make([]byte, 0xFFFF),
				connections: make(map[int]*conn),
				svr:         svr,
			}
			_ = loop.poller.AddRead(svr.ln.fd)
			svr.eventLoopGroup.register(loop)
		} else {
			return err
		}
	}
	svr.loopGroupSize = svr.eventLoopGroup.len()
	// Start loops in background
	svr.startLoops()
	return nil
}

func (svr *server) activateReactors(numLoops int) error {
	for i := 0; i < numLoops; i++ {
		if p, err := netpoll.OpenPoller(); err == nil {
			loop := &loop{
				idx:         i,
				poller:      p,
				packet:      make([]byte, 0xFFFF),
				connections: make(map[int]*conn),
				svr:         svr,
			}
			svr.eventLoopGroup.register(loop)
		} else {
			return err
		}
	}
	svr.loopGroupSize = svr.eventLoopGroup.len()
	// Start sub reactors...
	svr.startReactors()

	if p, err := netpoll.OpenPoller(); err == nil {
		loop := &loop{
			idx:    -1,
			poller: p,
			svr:    svr,
		}
		_ = loop.poller.AddRead(svr.ln.fd)
		svr.mainLoop = loop
		// Start main reactor...
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
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		sniffError(l.poller.Trigger(func() error {
			return ErrClosing
		}))
		return true
	})

	if svr.mainLoop != nil {
		sniffError(svr.mainLoop.poller.Trigger(func() error {
			return ErrClosing
		}))
	}

	// Wait on all loops to complete reading events
	svr.wg.Wait()

	// Close loops and all outstanding connections
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		for _, c := range l.connections {
			sniffError(l.loopCloseConn(c, nil))
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
	svr.eventLoopGroup = new(eventLoopGroup)
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.tch = make(chan time.Duration)
	svr.opts = options

	server := Server{options.Multicore, listener.lnaddr, numCPU}
	action := svr.eventHandler.OnInitComplete(server)
	switch action {
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

func (ln *listener) close() {
	if ln.f != nil {
		sniffError(ln.f.Close())
	}
	if ln.ln != nil {
		sniffError(ln.ln.Close())
	}
	if ln.pconn != nil {
		sniffError(ln.pconn.Close())
	}
	if ln.network == "unix" {
		sniffError(os.RemoveAll(ln.addr))
	}
}

// system takes the net listener and detaches it from it's parent
// event loop, grabs the file descriptor, and makes it non-blocking.
func (ln *listener) system() error {
	var err error
	switch netln := ln.ln.(type) {
	case nil:
		switch pconn := ln.pconn.(type) {
		case *net.UDPConn:
			ln.f, err = pconn.File()
		}
	case *net.TCPListener:
		ln.f, err = netln.File()
	case *net.UnixListener:
		ln.f, err = netln.File()
	}
	if err != nil {
		ln.close()
		return err
	}
	ln.fd = int(ln.f.Fd())
	return unix.SetNonblock(ln.fd, true)
}

func sniffError(err error) {
	if err != nil {
		log.Println(err)
	}
}
