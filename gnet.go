// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnet

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/gnet/internal/toolkit"
	"github.com/panjf2000/gnet/pkg/errors"
	"github.com/panjf2000/gnet/pkg/logging"
	"github.com/panjf2000/gnet/pkg/ringbuffer"
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

	// Addr is the listening address that align
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
		count += int(el.loadConn())
		return true
	})
	return
}

// DupFd returns a copy of the underlying file descriptor of listener.
// It is the caller's responsibility to close dupFD when finished.
// Closing listener does not affect dupFD, and closing dupFD does not affect listener.
func (s Server) DupFd() (dupFD int, err error) {
	dupFD, sc, err := s.svr.ln.dup()
	if err != nil {
		logging.Warnf("%s failed when duplicating new fd\n", sc)
	}
	return
}

// Conn is an interface of gnet connection.
type Conn interface {
	// ================================== Non-concurrency-safe API's ==================================

	// Context returns a user-defined context.
	Context() (ctx interface{})

	// SetContext sets a user-defined context.
	SetContext(ctx interface{})

	// LocalAddr is the connection's local socket address.
	LocalAddr() (addr net.Addr)

	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() (addr net.Addr)

	// Read reads all data from inbound ring-buffer without moving "read" pointer,
	// which means it does not evict the data from buffers actually and those data will
	// present in buffers until the ResetBuffer method is called.
	//
	// Note that the (buf []byte) returned by Read() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Read() (buf []byte)

	// ResetBuffer resets the buffers, which means all data in inbound ring-buffer and event-loop-buffer will be evicted.
	ResetBuffer()

	// ReadN reads bytes with the given length from inbound ring-buffer without moving "read" pointer,
	// which means it will not evict the data from buffers until the ShiftN method is called,
	// it reads data from the inbound ring-buffer and returns both bytes and the size of it.
	// If the length of the available data is less than the given "n", ReadN will return all available data,
	// so you should make use of the variable "size" returned by ReadN() to be aware of the exact length of the returned data.
	//
	// Note that the []byte buf returned by ReadN() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	ReadN(n int) (size int, buf []byte)

	// ShiftN shifts "read" pointer in the internal buffers with the given length.
	ShiftN(n int) (size int)

	// BufferLength returns the length of available data in the internal buffers.
	BufferLength() (size int)

	// ==================================== Concurrency-safe API's ====================================

	// SendTo writes data for UDP sockets, it allows you to send data back to UDP socket in individual goroutines.
	SendTo(buf []byte) error

	// AsyncWrite writes one byte slice to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWrite(buf []byte) error

	// AsyncWritev writes multiple byte slices to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWritev(bs [][]byte) error

	// Wake triggers a React event for the connection.
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
		// The parameter server has information and various utilities.
		OnInitComplete(server Server) (action Action)

		// OnShutdown fires when the server is being shut down, it is called right after
		// all event-loops and connections are closed.
		OnShutdown(server Server)

		// OnOpened fires when a new connection has been opened.
		// The Conn c has information about the connection such as it's local and remote address.
		// The parameter out is the return value which is going to be sent back to the peer.
		// It is usually not recommended to send large amounts of data back to the peer in OnOpened.
		//
		// Note that the bytes returned by OnOpened will be sent back to the peer without being encoded.
		OnOpened(c Conn) (out []byte, action Action)

		// OnClosed fires when a connection has been closed.
		// The parameter err is the last known connection error.
		OnClosed(c Conn, err error) (action Action)

		// PreWrite fires just before a packet is written to the peer socket, this event function is usually where
		// you put some code of logging/counting/reporting or any fore operations before writing data to the peer.
		PreWrite(c Conn)

		// AfterWrite fires right after a packet is written to the peer socket, this event function is usually where
		// you put the []byte returned from React() back to your memory pool.
		AfterWrite(c Conn, b []byte)

		// React fires when a socket receives data from the peer.
		// Call c.Read() or c.ReadN(n) of Conn c to read incoming data from the peer.
		// The parameter out is the return value which is going to be sent back to the peer.
		//
		// Note that the parameter packet returned from React() is not allowed to be passed to a new goroutine,
		// as this []byte will be reused within event-loop after React() returns.
		// If you have to use packet in a new goroutine, then you need to make a copy of buf and pass this copy
		// to that new goroutine.
		React(packet []byte, c Conn) (out []byte, action Action)

		// Tick fires immediately after the server starts and will fire again
		// following the duration specified by the delay return value.
		Tick() (delay time.Duration, action Action)
	}

	// EventServer is a built-in implementation of EventHandler which sets up each method with a default implementation,
	// you can compose it with your own implementation of EventHandler when you don't want to implement all methods
	// in EventHandler.
	EventServer struct{}
)

// OnInitComplete fires when the server is ready for accepting connections.
// The parameter server has information and various utilities.
func (es *EventServer) OnInitComplete(svr Server) (action Action) {
	return
}

// OnShutdown fires when the server is being shut down, it is called right after
// all event-loops and connections are closed.
func (es *EventServer) OnShutdown(svr Server) {
}

// OnOpened fires when a new connection has been opened.
// The parameter out is the return value which is going to be sent back to the peer.
func (es *EventServer) OnOpened(c Conn) (out []byte, action Action) {
	return
}

// OnClosed fires when a connection has been closed.
// The parameter err is the last known connection error.
func (es *EventServer) OnClosed(c Conn, err error) (action Action) {
	return
}

// PreWrite fires just before a packet is written to the peer socket, this event function is usually where
// you put some code of logging/counting/reporting or any fore operations before writing data to the peer.
func (es *EventServer) PreWrite(c Conn) {
}

// AfterWrite fires right after a packet is written to the peer socket, this event function is usually where
// you put the []byte's back to your memory pool.
func (es *EventServer) AfterWrite(c Conn, b []byte) {
}

// React fires when a connection sends the server data.
// Call c.Read() or c.ReadN(n) of Conn c to read incoming data from the peer.
// The parameter out is the return value which is going to be sent back to the peer.
func (es *EventServer) React(packet []byte, c Conn) (out []byte, action Action) {
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
//
//	tcp   - bind to both IPv4 and IPv6
//	tcp4  - IPv4
//	tcp6  - IPv6
//	udp   - bind to both IPv4 and IPv6
//	udp4  - IPv4
//	udp6  - IPv6
//	unix  - Unix Domain Socket
//
// The "tcp" network scheme is assumed when one is not specified.
func Serve(eventHandler EventHandler, protoAddr string, opts ...Option) (err error) {
	options := loadOptions(opts...)

	logging.Debugf("default logging level is %s", logging.LogLevel())

	var (
		logger logging.Logger
		flush  func() error
	)
	if options.LogPath != "" {
		if logger, flush, err = logging.CreateLoggerAsLocalFile(options.LogPath, options.LogLevel); err != nil {
			return
		}
	} else {
		logger = logging.GetDefaultLogger()
	}
	if options.Logger == nil {
		options.Logger = logger
	}
	defer func() {
		if flush != nil {
			_ = flush()
		}
		logging.Cleanup()
	}()

	// The maximum number of operating system threads that the Go program can use is initially set to 10000,
	// which should also be the maximum amount of I/O event-loops locked to OS threads that users can start up.
	if options.LockOSThread && options.NumEventLoop > 10000 {
		logging.Errorf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop)
		return errors.ErrTooManyEventLoopThreads
	}
	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = ringbuffer.MaxStreamBufferCap
	case rbc <= ringbuffer.DefaultBufferSize:
		options.ReadBufferCap = ringbuffer.DefaultBufferSize
	default:
		options.ReadBufferCap = toolkit.CeilToPowerOfTwo(rbc)
	}

	network, addr := parseProtoAddr(protoAddr)

	var ln *listener
	if ln, err = initListener(network, addr, options); err != nil {
		return
	}
	defer ln.close()

	return serve(eventHandler, ln, options, protoAddr)
}

var (
	allServers sync.Map

	// shutdownPollInterval is how often we poll to check whether server has been shut down during gnet.Stop().
	shutdownPollInterval = 500 * time.Millisecond
)

// Stop gracefully shuts down the server without interrupting any active event-loops,
// it waits indefinitely for connections and event-loops to be closed and then shuts down.
func Stop(ctx context.Context, protoAddr string) error {
	var svr *server
	if s, ok := allServers.Load(protoAddr); ok {
		svr = s.(*server)
		svr.signalShutdown()
		defer allServers.Delete(protoAddr)
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
