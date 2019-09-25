// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/panjf2000/gnet/netpoll"
)

var (
	ErrClosing  = errors.New("closing")
	ErrReactNil = errors.New("must set up Event.React()")
)

const connRingBufferSize = 1024

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota
	// Close closes the connection.
	Close
	// Shutdown shutdowns the server.
	Shutdown
)

// Server represents a server context which provides information about the
// running server and has control functions for managing state.
type Server struct {
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care with synchonizing memory between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of runtime.NumCPU().
	Multicore bool
	// The addrs parameter is an array of listening addresses that align
	// with the addr strings passed to the Serve function.
	Addr net.Addr
	// NumLoops is the number of loops that the server is using.
	NumLoops int
}

// Conn is an gnet connection.
type Conn interface {
	// Context returns a user-defined context.
	Context() interface{}
	// SetContext sets a user-defined context.
	SetContext(interface{})
	// LocalAddr is the connection's local socket address.
	LocalAddr() net.Addr
	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() net.Addr
	// Wake triggers a React event for this connection.
	Wake()
	// ReadPair reads all data from ring buffer.
	ReadPair() ([]byte, []byte)
	// ReadBytes reads all data and return a new slice.
	ReadBytes() []byte
	// AdvanceBuffer advances the read pointer of ring buffer.
	AdvanceBuffer(int)
	// ResetBuffer resets the ring buffer.
	ResetBuffer()
	// AyncWrite writes data asynchronously.
	AsyncWrite(buf []byte)
}

type eventLoopGrouper interface {
	register(*loop)
	next() *loop
	iterate(func(int, *loop) bool)
	len() int
}

// Events represents the server events for the Serve call.
// Each event has an Action return value that is used manage the state
// of the connection and server.
type Events struct {
	// OnInitComplete fires when the server can accept connections. The server
	// parameter has information and various utilities.
	OnInitComplete func(server Server) (action Action)
	// OnOpened fires when a new connection has opened.
	// The info parameter has information about the connection such as
	// it's local and remote address.
	// Use the out return value to write data to the connection.
	// The opts return value is used to set connection options.
	OnOpened func(c Conn) (out []byte, opts Options, action Action)
	// OnClosed fires when a connection has closed.
	// The err parameter is the last known connection error.
	OnClosed func(c Conn, err error) (action Action)
	// PreWrite fires just before any data is written to any client socket.
	PreWrite func()
	// React fires when a connection sends the server data.
	// The in parameter is the incoming data.
	// Use the out return value to write data to the connection.
	React func(c Conn) (out []byte, action Action)
	// Tick fires immediately after the server starts and will fire again
	// following the duration specified by the delay return value.
	Tick func() (delay time.Duration, action Action)
}

// Serve starts handling events for the specified addresses.
//
// Addresses should use a scheme prefix and be formatted
// like `tcp://192.168.0.10:9851` or `unix://socket`.
// Valid network schemes:
//  tcp   - bind to both IPv4 and IPv6
//  tcp4  - IPv4
//  tcp6  - IPv6
//  udp   - bind to both IPv4 and IPv6
//  udp4  - IPv4
//  udp6  - IPv6
//  unix  - Unix Domain Socket
//
// The "tcp" network scheme is assumed when one is not specified.
func Serve(events Events, addr string, opts ...Option) error {
	if events.React == nil {
		return ErrReactNil
	}

	var ln listener
	defer ln.close()

	options := initOptions(opts...)

	ln.network, ln.addr = parseAddr(addr)
	if ln.network == "unix" {
		sniffError(os.RemoveAll(ln.addr))
	}
	var err error
	if ln.network == "udp" {
		if options.ReusePort {
			ln.pconn, err = netpoll.ReusePortListenPacket(ln.network, ln.addr)
		} else {
			ln.pconn, err = net.ListenPacket(ln.network, ln.addr)
		}
	} else {
		if options.ReusePort {
			ln.ln, err = netpoll.ReusePortListen(ln.network, ln.addr)
		} else {
			ln.ln, err = net.Listen(ln.network, ln.addr)
		}
	}
	if err != nil {
		return err
	}
	if ln.pconn != nil {
		ln.lnaddr = ln.pconn.LocalAddr()
	} else {
		ln.lnaddr = ln.ln.Addr()
	}
	if err := ln.system(); err != nil {
		return err
	}
	return serve(events, &ln, options)
}

type listener struct {
	ln      net.Listener
	lnaddr  net.Addr
	pconn   net.PacketConn
	f       *os.File
	fd      int
	network string
	addr    string
}

func parseAddr(addr string) (network, address string) {
	network = "tcp"
	address = addr
	if strings.Contains(address, "://") {
		network = strings.Split(address, "://")[0]
		address = strings.Split(address, "://")[1]
	}
	return
}
