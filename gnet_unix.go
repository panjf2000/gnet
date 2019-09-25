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
	events         Events // user events
	mainLoop       *loop
	eventLoopGroup eventLoopGrouper
	loopGroupSize  int // number of loops
	opts           *Options
	ln             *listener          // all the listeners
	wg             sync.WaitGroup     // loop close waitgroup
	cond           *sync.Cond         // shutdown signaler
	tch            chan time.Duration // ticker channel
}

// waitForShutdown waits for a signal to shutdown
func (svr *server) waitForShutdown() {
	svr.cond.L.Lock()
	svr.cond.Wait()
	svr.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins server closing
func (svr *server) signalShutdown() {
	svr.cond.L.Lock()
	svr.cond.Signal()
	svr.cond.L.Unlock()
}

func (svr *server) startLoop(loop *loop) {
	svr.wg.Add(1)
	go func() {
		go loop.loopRun(svr)
		svr.wg.Done()
	}()
}

func (svr *server) activateLoops(numLoops int) {
	// Create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		loop := &loop{
			idx:         i,
			poller:      netpoll.OpenPoller(),
			packet:      make([]byte, 0xFFFF),
			connections: make(map[int]*conn),
		}
		loop.poller.AddRead(svr.ln.fd)
		svr.eventLoopGroup.register(loop)

		// Start loops in background
		svr.startLoop(loop)
	}
	svr.loopGroupSize = svr.eventLoopGroup.len()
}

func (svr *server) activateReactors(numLoops int) {
	svr.wg.Add(numLoops + 1)
	for i := 0; i < numLoops; i++ {
		loop := &loop{
			idx:         i,
			poller:      netpoll.OpenPoller(),
			packet:      make([]byte, 0xFFFF),
			connections: make(map[int]*conn),
		}
		svr.eventLoopGroup.register(loop)
	}
	svr.loopGroupSize = svr.eventLoopGroup.len()
	// Start sub reactors...
	svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
		go svr.activateSubReactor(l)
		return true
	})

	loop := &loop{
		idx:    -1,
		poller: netpoll.OpenPoller(),
	}
	if svr.ln.pconn != nil && loop.packet == nil {
		loop.packet = make([]byte, 0xFFFF)
	}
	loop.poller.AddRead(svr.ln.fd)
	svr.mainLoop = loop
	// Start main reactor...
	go svr.activateMainReactor()
}

func (svr *server) start(numCPU int) {
	if svr.opts.ReusePort {
		svr.activateLoops(numCPU)
	} else {
		svr.activateReactors(numCPU)
	}
}

func serve(events Events, listener *listener, options *Options) error {
	// Figure out the correct number of loops/goroutines to use.
	var numCPU int
	if options.Multicore {
		numCPU = runtime.NumCPU()
	} else {
		numCPU = 1
	}

	svr := new(server)
	svr.events = events
	svr.ln = listener
	svr.eventLoopGroup = new(eventLoopGroup)
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.tch = make(chan time.Duration)
	svr.opts = options

	if svr.events.OnInitComplete != nil {
		var server Server
		server.NumLoops = numCPU
		server.Addr = listener.lnaddr
		action := svr.events.OnInitComplete(server)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		// Wait on a signal for shutdown
		svr.waitForShutdown()

		// Notify all loops to close by closing all listeners
		svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
			sniffError(l.poller.Trigger(ErrClosing))
			return true
		})
		if svr.mainLoop != nil {
			sniffError(svr.mainLoop.poller.Trigger(ErrClosing))
		}

		// Wait on all loops to complete reading events
		svr.wg.Wait()

		// Close loops and all outstanding connections
		svr.eventLoopGroup.iterate(func(i int, l *loop) bool {
			for _, c := range l.connections {
				sniffError(l.loopCloseConn(svr, c, nil))
			}
			sniffError(l.poller.Close())
			return true
		})
		if svr.mainLoop != nil {
			sniffError(svr.mainLoop.poller.Close())
		}
	}()

	svr.start(numCPU)
	return nil
}

func (ln *listener) close() {
	//if ln.fd != 0 {
	//	sniffError(unix.Close(ln.fd))
	//}
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
