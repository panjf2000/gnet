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

	"golang.org/x/sys/unix"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/panjf2000/gnet/internal"
)

type server struct {
	events   Events // user events
	mainLoop *loop
	loops    []*loop            // all the loops
	numLoops int                // number of loops
	lns      []*listener        // all the listeners
	wg       sync.WaitGroup     // loop close waitgroup
	cond     *sync.Cond         // shutdown signaler
	tch      chan time.Duration // ticker channel
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

func serve(events Events, listeners []*listener, reusePort bool) error {
	// figure out the correct number of loops/goroutines to use.
	numLoops := events.NumLoops
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}

	svr := &server{}
	svr.events = events
	svr.lns = listeners
	svr.cond = sync.NewCond(&sync.Mutex{})
	svr.tch = make(chan time.Duration)

	if svr.events.OnInitComplete != nil {
		var server Server
		server.NumLoops = numLoops
		server.Addrs = make([]net.Addr, len(listeners))
		for i, ln := range listeners {
			server.Addrs[i] = ln.lnaddr
		}
		action := svr.events.OnInitComplete(server)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		// wait on a signal for shutdown
		svr.waitForShutdown()

		// notify all loops to close by closing all listeners
		for _, loop := range svr.loops {
			sniffError(loop.poller.Trigger(errClosing))
		}
		if svr.mainLoop != nil {
			sniffError(svr.mainLoop.poller.Trigger(errClosing))
		}

		// wait on all loops to complete reading events
		svr.wg.Wait()

		// close loops and all outstanding connections
		for _, loop := range svr.loops {
			for _, c := range loop.fdconns {
				sniffError(loop.loopCloseConn(svr, c, nil))
			}
			sniffError(loop.poller.Close())
		}
	}()

	if reusePort {
		activateLoops(svr, numLoops)
	} else {
		activateReactors(svr, numLoops)
	}
	return nil
}

func activateLoops(svr *server, numLoops int) {
	// create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		loop := &loop{
			idx:     i,
			poller:  internal.OpenPoller(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*conn),
		}
		for _, ln := range svr.lns {
			loop.poller.AddRead(ln.fd)
		}
		svr.loops = append(svr.loops, loop)
	}
	svr.numLoops = len(svr.loops)
	// start loops in background
	svr.wg.Add(svr.numLoops)
	for _, loop := range svr.loops {
		go loop.loopRun(svr)
	}
}

func activateReactors(svr *server, numLoops int) {
	if numLoops == 1 {
		numLoops = 2
	}
	svr.wg.Add(numLoops)
	for i := 0; i < numLoops-1; i++ {
		loop := &loop{
			idx:     i,
			poller:  internal.OpenPoller(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*conn),
		}
		svr.loops = append(svr.loops, loop)
	}
	svr.numLoops = len(svr.loops)
	// start sub reactors...
	for _, loop := range svr.loops {
		go activateSubReactor(svr, loop)
	}

	loop := &loop{
		idx:    -1,
		poller: internal.OpenPoller(),
	}
	for _, ln := range svr.lns {
		loop.poller.AddRead(ln.fd)
	}
	svr.mainLoop = loop
	// start main reactor...
	go activateMainReactor(svr)
}

func (ln *listener) close() {
	if ln.fd != 0 {
		sniffError(unix.Close(ln.fd))
	}
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

func reuseportListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return reuseport.ListenPacket(proto, addr)
}

func reuseportListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}
