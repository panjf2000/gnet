// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/panjf2000/gnet/internal/netpoll"
)

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

var defaultLogger = Logger(log.New(os.Stderr, "", log.LstdFlags))

// Logger is used for logging formatted messages.
type Logger interface {
	// Printf must have the same semantics as log.Printf.
	Printf(format string, args ...interface{})
}

// Server represents a server context which provides information about the
// running server and has control functions for managing state.
type Server struct {
	// svr is the internal server struct.
	svr *server
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care of synchronizing the shared data between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of runtime.NumCPU().
	Multicore bool

	// The Addr parameter is the listening address that align
	// with the addr string passed to the Serve function.
	Addr net.Addr

	// NumEventLoop is the number of event-loops that the server is using.
	NumEventLoop int

	// ReusePort indicates whether SO_REUSEPORT is enable.
	ReusePort bool

	// TCPKeepAlive (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration
}

// CountConnections counts the number of currently active connections and returns it.
func (s Server) CountConnections() (count int) {
	s.svr.subLoopGroup.iterate(func(i int, el *eventloop) bool {
		count += int(el.loadConnCount())
		return true
	})
	return
}

// Conn is a interface of gnet connection.
type Conn interface {
	// Context returns a user-defined context.
	Context() (ctx interface{})

	// SetContext sets a user-defined context.
	SetContext(ctx interface{})

	// LocalAddr is the connection's local socket address.
	LocalAddr() (addr net.Addr)

	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() (addr net.Addr)

	// Read reads all data from inbound ring-buffer and event-loop-buffer without moving "read" pointer, which means
	// it does not evict the data from buffers actually and those data will present in buffers until the
	// ResetBuffer method is invoked.
	Read() (buf []byte)

	// ResetBuffer resets the buffers, which means all data in inbound ring-buffer and event-loop-buffer
	// will be evicted.
	ResetBuffer()

	// ReadN reads bytes with the given length from inbound ring-buffer and event-loop-buffer without moving
	// "read" pointer, which means it will not evict the data from buffers until the ShiftN method is invoked,
	// it reads data from the inbound ring-buffer and event-loop-buffer and returns the size of bytes.
	// If the length of the available data is less than the given "n", ReadN will returns all available data, so you
	// should make use of the variable "size" returned by it to be aware of the exact length of the returned data.
	ReadN(n int) (size int, buf []byte)

	// ShiftN shifts "read" pointer in buffers with the given length.
	ShiftN(n int) (size int)

	// BufferLength returns the length of available data in the inbound ring-buffer.
	BufferLength() (size int)

	// InboundBuffer returns the inbound ring-buffer.
	//InboundBuffer() *ringbuffer.RingBuffer

	// SendTo writes data for UDP sockets, it allows you to send data back to UDP socket in individual goroutines.
	SendTo(buf []byte) error

	// AsyncWrite writes data to client/connection asynchronously, usually you would invoke it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWrite(buf []byte) error

	// Wake triggers a React event for this connection.
	Wake() error

	// Close closes the current connection.
	Close() error
}

type (
	// EventHandler represents the server events' callbacks for the Serve call.
	// Each event has an Action return value that is used manage the state
	// of the connection and server.
	EventHandler interface {
		// OnInitComplete fires when the server is ready for accepting connections.
		// The server parameter has information and various utilities.
		OnInitComplete(server Server) (action Action)

		// OnOpened fires when a new connection has been opened.
		// The info parameter has information about the connection such as
		// it's local and remote address.
		// Use the out return value to write data to the connection.
		OnOpened(c Conn) (out []byte, action Action)

		// OnClosed fires when a connection has been closed.
		// The err parameter is the last known connection error.
		OnClosed(c Conn, err error) (action Action)

		// PreWrite fires just before any data is written to any client socket.
		PreWrite()

		// React fires when a connection sends the server data.
		// Invoke c.Read() or c.ReadN(n) within the parameter c to read incoming data from client/connection.
		// Use the out return value to write data to the client/connection.
		React(frame []byte, c Conn) (out []byte, action Action)

		// Tick fires immediately after the server starts and will fire again
		// following the duration specified by the delay return value.
		Tick() (delay time.Duration, action Action)
	}

	// EventServer is a built-in implementation of EventHandler which sets up each method with a default implementation,
	// you can compose it with your own implementation of EventHandler when you don't want to implement all methods
	// in EventHandler.
	EventServer struct {
	}
)

// OnInitComplete fires when the server is ready for accepting connections.
// The server parameter has information and various utilities.
func (es *EventServer) OnInitComplete(svr Server) (action Action) {
	return
}

// OnOpened fires when a new connection has been opened.
// The info parameter has information about the connection such as
// it's local and remote address.
// Use the out return value to write data to the connection.
func (es *EventServer) OnOpened(c Conn) (out []byte, action Action) {
	return
}

// OnClosed fires when a connection has been closed.
// The err parameter is the last known connection error.
func (es *EventServer) OnClosed(c Conn, err error) (action Action) {
	return
}

// PreWrite fires just before any data is written to any client socket.
func (es *EventServer) PreWrite() {
}

// React fires when a connection sends the server data.
// Invoke c.Read() or c.ReadN(n) within the parameter c to read incoming data from client/connection.
// Use the out return value to write data to the client/connection.
func (es *EventServer) React(frame []byte, c Conn) (out []byte, action Action) {
	return
}

// Tick fires immediately after the server starts and will fire again
// following the duration specified by the delay return value.
func (es *EventServer) Tick() (delay time.Duration, action Action) {
	return
}

// Serve starts handling events for the specified address.
//
// Address should use a scheme prefix and be formatted
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
func Serve(eventHandler EventHandler, addr string, opts ...Option) error {
	var ln listener
	defer func() {
		ln.close()
		if ln.network == "unix" {
			sniffErrorAndLog(os.RemoveAll(ln.addr))
		}
	}()

	options := loadOptions(opts...)

	if options.Logger != nil {
		defaultLogger = options.Logger
	}

	ln.network, ln.addr = parseAddr(addr)
	if ln.network == "unix" {
		sniffErrorAndLog(os.RemoveAll(ln.addr))
		if runtime.GOOS == "windows" {
			return ErrProtocolNotSupported
		}
	}
	var err error
	if ln.network == "udp" {
		if options.ReusePort && runtime.GOOS != "windows" {
			ln.pconn, err = netpoll.ReusePortListenPacket(ln.network, ln.addr)
		} else {
			ln.pconn, err = net.ListenPacket(ln.network, ln.addr)
		}
	} else {
		if options.ReusePort && runtime.GOOS != "windows" {
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
	return serve(eventHandler, &ln, options)
}

func parseAddr(addr string) (network, address string) {
	network = "tcp"
	address = addr
	if strings.Contains(address, "://") {
		parts := strings.Split(address, "://")
		network = parts[0]
		address = parts[1]
	}
	return
}

func sniffErrorAndLog(err error) {
	if err != nil {
		defaultLogger.Printf(err.Error())
	}
}
