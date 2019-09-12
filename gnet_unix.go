// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build darwin netbsd freebsd openbsd dragonfly linux

package gnet

import (
	"fmt"
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
func (s *server) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins server closing
func (s *server) signalShutdown() {
	s.cond.L.Lock()
	s.cond.Signal()
	s.cond.L.Unlock()
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

	s := &server{}
	s.events = events
	s.lns = listeners
	s.cond = sync.NewCond(&sync.Mutex{})
	s.tch = make(chan time.Duration)

	if s.events.OnInitComplete != nil {
		var svr Server
		svr.NumLoops = numLoops
		svr.Addrs = make([]net.Addr, len(listeners))
		for i, ln := range listeners {
			svr.Addrs[i] = ln.lnaddr
		}
		action := s.events.OnInitComplete(svr)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		// wait on a signal for shutdown
		s.waitForShutdown()

		// notify all loops to close by closing all listeners
		for _, l := range s.loops {
			sniffError(l.poll.Trigger(errClosing))
		}
		if s.mainLoop != nil {
			sniffError(s.mainLoop.poll.Trigger(errClosing))
		}

		// wait on all loops to complete reading events
		s.wg.Wait()

		// close loops and all outstanding connections
		for _, l := range s.loops {
			for _, c := range l.fdconns {
				sniffError(loopCloseConn(s, l, c, nil))
			}
			sniffError(l.poll.Close())
		}
	}()

	if reusePort {
		activateLoops(s, numLoops)
	} else {
		activateReactors(s, numLoops)
	}
	return nil
}

func activateLoops(s *server, numLoops int) {
	// create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		l := &loop{
			idx:     i,
			poll:    internal.OpenPoll(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*conn),
		}
		for _, ln := range s.lns {
			l.poll.AddRead(ln.fd)
		}
		s.loops = append(s.loops, l)
	}
	s.numLoops = len(s.loops)
	// start loops in background
	s.wg.Add(s.numLoops)
	for _, l := range s.loops {
		go loopRun(s, l)
	}
}

func activateReactors(s *server, numLoops int) {
	if numLoops == 1 {
		numLoops = 2
	}
	s.wg.Add(numLoops)

	l := &loop{
		idx:  -1,
		poll: internal.OpenPoll(),
	}
	fmt.Printf("main reactor epoll: %d\n", l.poll.GetFD())
	for _, ln := range s.lns {
		l.poll.AddRead(ln.fd)
	}
	s.mainLoop = l
	// start main reactor...
	go activateMainReactor(s)

	for i := 0; i < numLoops-1; i++ {
		l := &loop{
			idx:     i,
			poll:    internal.OpenPoll(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*conn),
		}
		fmt.Printf("sub reactor: %d epoll: %d\n", i, l.poll.GetFD())
		s.loops = append(s.loops, l)
	}
	s.numLoops = len(s.loops)
	// start sub reactors...
	for _, l := range s.loops {
		go activateSubReactor(s, l)
	}
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
