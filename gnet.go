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
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/internal/logging"
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

// Server represents a server context which provides information about the
// running server and has control functions for managing state.
type Server struct {
	// svr is the internal server struct.
	svr *server
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care of synchronizing the shared data between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of logical CPUs usable by the current process.
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
	s.svr.lb.iterate(func(i int, el *eventloop) bool {
		count += int(atomic.LoadInt32(&el.connCount))
		return true
	})
	return
}

// DupFd returns a copy of the underlying file descriptor of listener.
// It is the caller's responsibility to close dupFD when finished.
// Closing listener does not affect dupFD, and closing dupFD does not affect listener.
func (s Server) DupFd() (dupFD int, err error) {
	dupFD, sc, err := s.svr.ln.Dup()
	if err != nil {
		logging.DefaultLogger.Warnf("%s failed when duplicating new fd\n", sc)
	}
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
	// ResetBuffer method is called.
	Read() (buf []byte)

	// ResetBuffer resets the buffers, which means all data in inbound ring-buffer and event-loop-buffer will be evicted.
	ResetBuffer()

	// ReadN reads bytes with the given length from inbound ring-buffer and event-loop-buffer without moving
	// "read" pointer, which means it will not evict the data from buffers until the ShiftN method is called,
	// it reads data from the inbound ring-buffer and event-loop-buffer and returns both bytes and the size of it.
	// If the length of the available data is less than the given "n", ReadN will return all available data, so you
	// should make use of the variable "size" returned by it to be aware of the exact length of the returned data.
	ReadN(n int) (size int, buf []byte)

	// ShiftN shifts "read" pointer in the internal buffers with the given length.
	ShiftN(n int) (size int)

	// BufferLength returns the length of available data in the internal buffers.
	BufferLength() (size int)

	// InboundBuffer returns the inbound ring-buffer.
	// InboundBuffer() *ringbuffer.RingBuffer

	// SendTo writes data for UDP sockets, it allows you to send data back to UDP socket in individual goroutines.
	SendTo(buf []byte) error

	// AsyncWrite writes data to client/connection asynchronously, usually you would call it in individual goroutines
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
		// The parameter:server has information and various utilities.
		OnInitComplete(server Server) (action Action)

		// OnShutdown fires when the server is being shut down, it is called right after
		// all event-loops and connections are closed.
		OnShutdown(server Server)

		// OnOpened fires when a new connection has been opened.
		// The parameter:c has information about the connection such as it's local and remote address.
		// Parameter:out is the return value which is going to be sent back to the client.
		OnOpened(c Conn) (out []byte, action Action)

		// OnClosed fires when a connection has been closed.
		// The parameter:err is the last known connection error.
		OnClosed(c Conn, err error) (action Action)

		// PreWrite fires just before any data is written to any client socket, this event function is usually used to
		// put some code of logging/counting/reporting or any prepositive operations before writing data to client.
		PreWrite()

		// React fires when a connection sends the server data.
		// Call c.Read() or c.ReadN(n) within the parameter:c to read incoming data from client.
		// Parameter:out is the return value which is going to be sent back to the client.
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
// The parameter:server has information and various utilities.
func (es *EventServer) OnInitComplete(svr Server) (action Action) {
	return
}

// OnShutdown fires when the server is being shut down, it is called right after
// all event-loops and connections are closed.
func (es *EventServer) OnShutdown(svr Server) {
}

// OnOpened fires when a new connection has been opened.
// The parameter:c has information about the connection such as it's local and remote address.
// Parameter:out is the return value which is going to be sent back to the client.
func (es *EventServer) OnOpened(c Conn) (out []byte, action Action) {
	return
}

// OnClosed fires when a connection has been closed.
// The parameter:err is the last known connection error.
func (es *EventServer) OnClosed(c Conn, err error) (action Action) {
	return
}

// PreWrite fires just before any data is written to any client socket, this event function is usually used to
// put some code of logging/counting/reporting or any prepositive operations before writing data to client.
func (es *EventServer) PreWrite() {
}

// React fires when a connection sends the server data.
// Call c.Read() or c.ReadN(n) within the parameter:c to read incoming data from client.
// Parameter:out is the return value which is going to be sent back to the client.
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
func Serve(eventHandler EventHandler, protoAddr string, opts ...Option) (err error) {
	options := loadOptions(opts...)

	if options.Logger != nil {
		logging.DefaultLogger = options.Logger
	}
	defer logging.Cleanup()

	// The maximum number of operating system threads that the Go program can use is initially set to 10000,
	// which should be the maximum amount of I/O event-loops locked to OS threads users can start up.
	if options.LockOSThread && options.NumEventLoop > 10000 {
		logging.DefaultLogger.Errorf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop)
		return errors.ErrTooManyEventLoopThreads
	}

	network, addr := parseProtoAddr(protoAddr)

	var ln *listener
	if ln, err = initListener(network, addr, options.ReusePort); err != nil {
		return
	}
	defer ln.close()

	return serve(eventHandler, ln, options, protoAddr)
}

// shutdownPollInterval is how often we poll to check whether server has been shut down during gnet.Stop().
var shutdownPollInterval = 500 * time.Millisecond

// Stop gracefully shuts down the server without interrupting any active eventloops,
// it waits indefinitely for connections and eventloops to be closed and then shuts down.
func Stop(ctx context.Context, protoAddr string) error {
	var svr *server
	if s, ok := serverFarm.Load(protoAddr); ok {
		svr = s.(*server)
		svr.signalShutdown()
		defer serverFarm.Delete(protoAddr)
	} else {
		return errors.ErrServerInShutdown
	}

	if svr.isInShutdown() {
		return errors.ErrServerInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if svr.isInShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func parseProtoAddr(addr string) (network, address string) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
	}
	return
}

func sniffErrorAndLog(err error) {
	if err != nil {
		logging.DefaultLogger.Errorf(err.Error())
	}
}

// channelBuffer determines whether the channel should be a buffered channel to get the best performance.
var channelBuffer = func() int {
	// Use blocking channel if GOMAXPROCS=1.
	// This switches context from sender to receiver immediately,
	// which results in higher performance.
	var n int
	if n = runtime.GOMAXPROCS(0); n == 1 {
		return 0
	}

	// Make channel non-blocking and set up its capacity with GOMAXPROCS if GOMAXPROCS>1,
	// otherwise the sender might be dragged down if the receiver is CPU-bound.
	//
	// GOMAXPROCS determines how many goroutines can run in parallel,
	// which makes it the best choice as the channel capacity,
	return n
}()
