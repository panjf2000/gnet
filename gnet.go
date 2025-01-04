// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

// Package gnet implements a high-performance, lightweight, non-blocking,
// event-driven networking framework written in pure Go.
//
// Visit https://gnet.host/ for more details about gnet.
package gnet

import (
	"context"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/math"
)

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota

	// Close closes the connection.
	Close

	// Shutdown shutdowns the engine.
	Shutdown
)

// Engine represents an engine context which provides some functions.
type Engine struct {
	// eng is the internal engine struct.
	eng *engine
}

// Validate checks whether the engine is available.
func (e Engine) Validate() error {
	if e.eng == nil || len(e.eng.listeners) == 0 {
		return errors.ErrEmptyEngine
	}
	if e.eng.isShutdown() {
		return errors.ErrEngineInShutdown
	}
	return nil
}

// CountConnections counts the number of currently active connections and returns it.
func (e Engine) CountConnections() (count int) {
	if e.Validate() != nil {
		return -1
	}

	e.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		count += int(el.countConn())
		return true
	})
	return
}

// Dup returns a copy of the underlying file descriptor of listener.
// It is the caller's responsibility to close dupFD when finished.
// Closing listener does not affect dupFD, and closing dupFD does not affect listener.
func (e Engine) Dup() (fd int, err error) {
	if err := e.Validate(); err != nil {
		return -1, err
	}
	if len(e.eng.listeners) > 1 {
		return -1, errors.ErrUnsupportedOp
	}
	for _, ln := range e.eng.listeners {
		fd, err = ln.dup()
	}
	return
}

// Stop gracefully shuts down this Engine without interrupting any active event-loops,
// it waits indefinitely for connections and event-loops to be closed and then shuts down.
func (e Engine) Stop(ctx context.Context) error {
	if err := e.Validate(); err != nil {
		return err
	}

	e.eng.shutdown(nil)

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if e.eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

/*
type asyncCmdType uint8

const (
	asyncCmdClose = iota + 1
	asyncCmdWake
	asyncCmdWrite
	asyncCmdWritev
)

type asyncCmd struct {
	fd  gfd.GFD
	typ asyncCmdType
	cb  AsyncCallback
	param any
}

// AsyncWrite writes data to the given connection asynchronously.
func (e Engine) AsyncWrite(fd gfd.GFD, p []byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWrite, cb: cb, param: p}, false)
}

// AsyncWritev is like AsyncWrite, but it accepts a slice of byte slices.
func (e Engine) AsyncWritev(fd gfd.GFD, batch [][]byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWritev, cb: cb, param: batch}, false)
}

// Close closes the given connection.
func (e Engine) Close(fd gfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdClose, cb: cb}, false)
}

// Wake wakes up the given connection.
func (e Engine) Wake(fd gfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWake, cb: cb}, true)
}
*/

// Reader is an interface that consists of a number of methods for reading that Conn must implement.
//
// Note that the methods in this interface are not concurrency-safe for concurrent use,
// you must invoke them within any method in EventHandler.
type Reader interface {
	io.Reader
	io.WriterTo

	// Next returns a slice containing the next n bytes from the buffer,
	// advancing the buffer as if the bytes had been returned by Read.
	// Calling this method has the same effect as calling Peek and Discard.
	// If the amount of the available bytes is less than requested, a pair of (0, io.ErrShortBuffer)
	// is returned.
	//
	// Note that the []byte buf returned by Next() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Next(n int) (buf []byte, err error)

	// Peek returns the next n bytes without advancing the inbound buffer, the returned bytes
	// remain valid until a Discard is called. If the amount of the available bytes is
	// less than requested, a pair of (0, io.ErrShortBuffer) is returned.
	//
	// Note that the []byte buf returned by Peek() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Peek(n int) (buf []byte, err error)

	// Discard advances the inbound buffer with next n bytes, returning the number of bytes discarded.
	Discard(n int) (discarded int, err error)

	// InboundBuffered returns the number of bytes that can be read from the current buffer.
	InboundBuffered() (n int)
}

// Writer is an interface that consists of a number of methods for writing that Conn must implement.
type Writer interface {
	io.Writer     // not concurrency-safe
	io.ReaderFrom // not concurrency-safe

	// Writev writes multiple byte slices to remote synchronously, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	Writev(bs [][]byte) (n int, err error)

	// Flush writes any buffered data to the underlying connection, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	Flush() (err error)

	// OutboundBuffered returns the number of bytes that can be read from the current buffer.
	// it's not concurrency-safe, you must invoke it within any method in EventHandler.
	OutboundBuffered() (n int)

	// AsyncWrite writes bytes to remote asynchronously, it's concurrency-safe,
	// you don't have to invoke it within any method in EventHandler,
	// usually you would call it in an individual goroutine.
	//
	// Note that it will go synchronously with UDP, so it is needless to call
	// this asynchronous method, we may disable this method for UDP and just
	// return ErrUnsupportedOp in the future, therefore, please don't rely on
	// this method to do something important under UDP, if you're working with UDP,
	// just call Conn.Write to send back your data.
	AsyncWrite(buf []byte, callback AsyncCallback) (err error)

	// AsyncWritev writes multiple byte slices to remote asynchronously,
	// you don't have to invoke it within any method in EventHandler,
	// usually you would call it in an individual goroutine.
	AsyncWritev(bs [][]byte, callback AsyncCallback) (err error)
}

// AsyncCallback is a callback that will be invoked after the asynchronous function finishes.
//
// Note that the parameter gnet.Conn might have been already released when it's UDP protocol,
// thus it shouldn't be accessed.
// This callback will be executed in event-loop, thus it must not block, otherwise,
// it blocks the event-loop.
type AsyncCallback func(c Conn, err error) error

// Socket is a set of functions which manipulate the underlying file descriptor of a connection.
//
// Note that the methods in this interface are concurrency-safe for concurrent use,
// you don't have to invoke them within any method in EventHandler.
type Socket interface {
	// Gfd returns the gfd of socket.
	// Gfd() gfd.GFD

	// Fd returns the underlying file descriptor.
	Fd() int

	// Dup returns a copy of the underlying file descriptor.
	// It is the caller's responsibility to close fd when finished.
	// Closing c does not affect fd, and closing fd does not affect c.
	//
	// The returned file descriptor is different from the
	// connection's. Attempting to change properties of the original
	// using this duplicate may or may not have the desired effect.
	Dup() (int, error)

	// SetReadBuffer sets the size of the operating system's
	// receive buffer associated with the connection.
	SetReadBuffer(bytes int) error

	// SetWriteBuffer sets the size of the operating system's
	// transmit buffer associated with the connection.
	SetWriteBuffer(bytes int) error

	// SetLinger sets the behavior of Close on a connection which still
	// has data waiting to be sent or to be acknowledged.
	//
	// If sec < 0 (the default), the operating system finishes sending the
	// data in the background.
	//
	// If sec == 0, the operating system discards any unsent or
	// unacknowledged data.
	//
	// If sec > 0, the data is sent in the background as with sec < 0. On
	// some operating systems after sec seconds have elapsed any remaining
	// unsent data may be discarded.
	SetLinger(sec int) error

	// SetKeepAlivePeriod tells operating system to send keep-alive messages on the connection
	// and sets period between TCP keep-alive probes.
	SetKeepAlivePeriod(d time.Duration) error

	// SetNoDelay controls whether the operating system should delay
	// packet transmission in hopes of sending fewer packets (Nagle's
	// algorithm).
	// The default is true (no delay), meaning that data is sent as soon as possible after a Write.
	SetNoDelay(noDelay bool) error
}

// Conn is an interface of underlying connection.
type Conn interface {
	Reader // all methods in Reader are not concurrency-safe.
	Writer // some methods in Writer are concurrency-safe, some are not.
	Socket // all methods in Socket are concurrency-safe.

	// Context returns a user-defined context, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	Context() (ctx any)

	// SetContext sets a user-defined context, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	SetContext(ctx any)

	// LocalAddr is the connection's local socket address, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	LocalAddr() (addr net.Addr)

	// RemoteAddr is the connection's remote address, it's not concurrency-safe,
	// you must invoke it within any method in EventHandler.
	RemoteAddr() (addr net.Addr)

	// Wake triggers a OnTraffic event for the current connection, it's concurrency-safe.
	Wake(callback AsyncCallback) (err error)

	// CloseWithCallback closes the current connection, it's concurrency-safe.
	// Usually you should provide a non-nil callback for this method,
	// otherwise your better choice is Close().
	CloseWithCallback(callback AsyncCallback) (err error)

	// Close closes the current connection, implements net.Conn, it's concurrency-safe.
	Close() (err error)

	// SetDeadline implements net.Conn.
	SetDeadline(t time.Time) (err error)

	// SetReadDeadline implements net.Conn.
	SetReadDeadline(t time.Time) (err error)

	// SetWriteDeadline implements net.Conn.
	SetWriteDeadline(t time.Time) (err error)
}

type (
	// EventHandler represents the engine events' callbacks for the Run call.
	// Each event has an Action return value that is used manage the state
	// of the connection and engine.
	EventHandler interface {
		// OnBoot fires when the engine is ready for accepting connections.
		// The parameter engine has information and various utilities.
		OnBoot(eng Engine) (action Action)

		// OnShutdown fires when the engine is being shut down, it is called right after
		// all event-loops and connections are closed.
		OnShutdown(eng Engine)

		// OnOpen fires when a new connection has been opened.
		//
		// The Conn c has information about the connection such as its local and remote addresses.
		// The parameter out is the return value which is going to be sent back to the remote.
		// Sending large amounts of data back to the remote in OnOpen is usually not recommended.
		OnOpen(c Conn) (out []byte, action Action)

		// OnClose fires when a connection has been closed.
		// The parameter err is the last known connection error.
		OnClose(c Conn, err error) (action Action)

		// OnTraffic fires when a socket receives data from the remote.
		//
		// Note that the []byte returned from Conn.Peek(int)/Conn.Next(int) is not allowed to be passed to a new goroutine,
		// as this []byte will be reused within event-loop after OnTraffic() returns.
		// If you have to use this []byte in a new goroutine, you should either make a copy of it or call Conn.Read([]byte)
		// to read data into your own []byte, then pass the new []byte to the new goroutine.
		OnTraffic(c Conn) (action Action)

		// OnTick fires immediately after the engine starts and will fire again
		// following the duration specified by the delay return value.
		OnTick() (delay time.Duration, action Action)
	}

	// BuiltinEventEngine is a built-in implementation of EventHandler which sets up each method with a default implementation,
	// you can compose it with your own implementation of EventHandler when you don't want to implement all methods
	// in EventHandler.
	BuiltinEventEngine struct{}
)

// OnBoot fires when the engine is ready for accepting connections.
// The parameter engine has information and various utilities.
func (*BuiltinEventEngine) OnBoot(_ Engine) (action Action) {
	return
}

// OnShutdown fires when the engine is being shut down, it is called right after
// all event-loops and connections are closed.
func (*BuiltinEventEngine) OnShutdown(_ Engine) {
}

// OnOpen fires when a new connection has been opened.
// The parameter out is the return value which is going to be sent back to the remote.
func (*BuiltinEventEngine) OnOpen(_ Conn) (out []byte, action Action) {
	return
}

// OnClose fires when a connection has been closed.
// The parameter err is the last known connection error.
func (*BuiltinEventEngine) OnClose(_ Conn, _ error) (action Action) {
	return
}

// OnTraffic fires when a local socket receives data from the remote.
func (*BuiltinEventEngine) OnTraffic(_ Conn) (action Action) {
	return
}

// OnTick fires immediately after the engine starts and will fire again
// following the duration specified by the delay return value.
func (*BuiltinEventEngine) OnTick() (delay time.Duration, action Action) {
	return
}

// MaxStreamBufferCap is the default buffer size for each stream-oriented connection(TCP/Unix).
var MaxStreamBufferCap = 64 * 1024 // 64KB

func createListeners(addrs []string, opts ...Option) ([]*listener, *Options, error) {
	options := loadOptions(opts...)

	logger, logFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	if options.Logger == nil {
		if options.LogPath != "" {
			logger, logFlusher, _ = logging.CreateLoggerAsLocalFile(options.LogPath, options.LogLevel)
		}
		options.Logger = logger
	} else {
		logger = options.Logger
		logFlusher = nil
	}
	logging.SetDefaultLoggerAndFlusher(logger, logFlusher)

	logging.Debugf("default logging level is %s", logging.LogLevel())

	// The maximum number of operating system threads that the Go program can use is initially set to 10000,
	// which should also be the maximum amount of I/O event-loops locked to OS threads that users can start up.
	if options.LockOSThread && options.NumEventLoop > 10000 {
		logging.Errorf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop)
		return nil, nil, errors.ErrTooManyEventLoopThreads
	}

	if options.EdgeTriggeredIOChunk > 0 {
		options.EdgeTriggeredIO = true
		options.EdgeTriggeredIOChunk = math.CeilToPowerOfTwo(options.EdgeTriggeredIOChunk)
	} else if options.EdgeTriggeredIO {
		options.EdgeTriggeredIOChunk = 1 << 20 // 1MB
	}

	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= ring.DefaultBufferSize:
		options.ReadBufferCap = ring.DefaultBufferSize
	default:
		options.ReadBufferCap = math.CeilToPowerOfTwo(rbc)
	}
	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= ring.DefaultBufferSize:
		options.WriteBufferCap = ring.DefaultBufferSize
	default:
		options.WriteBufferCap = math.CeilToPowerOfTwo(wbc)
	}

	var hasUDP, hasUnix bool
	for _, addr := range addrs {
		proto, _, err := parseProtoAddr(addr)
		if err != nil {
			return nil, nil, err
		}
		hasUDP = hasUDP || strings.HasPrefix(proto, "udp")
		hasUnix = hasUnix || proto == "unix"
	}

	// SO_REUSEPORT enables duplicate address and port bindings across various
	// Unix-like OSs, whereas there is platform-specific inconsistency:
	// Linux implemented SO_REUSEPORT with load balancing for incoming connections
	// while *BSD implemented it for only binding to the same address and port, which
	// makes it pointless to enable SO_REUSEPORT on *BSD and Darwin for gnet with
	// multiple event-loops because only the first or last event-loop will be constantly
	// woken up to accept incoming connections and handle I/O events while the rest of
	// event-loops remain idle.
	// Thus, we disable SO_REUSEPORT on *BSD and Darwin by default.
	//
	// Note that FreeBSD 12 introduced a new socket option named SO_REUSEPORT_LB
	// with the capability of load balancing, it's the equivalent of Linux's SO_REUSEPORT.
	// Also note that DragonFlyBSD 3.6.0 extended SO_REUSEPORT to distribute workload to
	// available sockets, which make it the same as Linux's SO_REUSEPORT.
	//
	// Despite the fact that SO_REUSEPORT can be set on a Unix domain socket
	// via setsockopt() without reporting an error, SO_REUSEPORT is actually
	// not supported for sockets of AF_UNIX. Thus, we avoid setting it on the
	// Unix domain sockets.
	goos := runtime.GOOS
	if (options.Multicore || options.NumEventLoop > 1) && options.ReusePort &&
		((goos != "linux" && goos != "dragonfly" && goos != "freebsd") || hasUnix) {
		options.ReusePort = false
	}

	// If there is UDP address in the list, we have no choice but to enable SO_REUSEPORT anyway,
	// also disable edge-triggered I/O for UDP by default.
	if hasUDP {
		options.ReusePort = true
		options.EdgeTriggeredIO = false
	}

	listeners := make([]*listener, len(addrs))
	for i, a := range addrs {
		proto, addr, err := parseProtoAddr(a)
		if err != nil {
			return nil, nil, err
		}
		ln, err := initListener(proto, addr, options)
		if err != nil {
			return nil, nil, err
		}
		listeners[i] = ln
	}

	return listeners, options, nil
}

// Run starts handling events on the specified address.
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
func Run(eventHandler EventHandler, protoAddr string, opts ...Option) error {
	listeners, options, err := createListeners([]string{protoAddr}, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
		logging.Cleanup()
	}()
	return run(eventHandler, listeners, options, []string{protoAddr})
}

// Rotate is like Run but accepts multiple network addresses.
func Rotate(eventHandler EventHandler, addrs []string, opts ...Option) error {
	listeners, options, err := createListeners(addrs, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
		logging.Cleanup()
	}()
	return run(eventHandler, listeners, options, addrs)
}

var (
	allEngines sync.Map

	// shutdownPollInterval is how often we poll to check whether engine has been shut down during gnet.Stop().
	shutdownPollInterval = 500 * time.Millisecond
)

// Stop gracefully shuts down the engine without interrupting any active event-loops,
// it waits indefinitely for connections and event-loops to be closed and then shuts down.
//
// Deprecated: The global Stop only shuts down the last registered Engine with the same
// protocol and IP:Port as the previous Engine's, which can lead to leaks of Engine if
// you invoke gnet.Run multiple times using the same protocol and IP:Port under the
// condition that WithReuseAddr(true) and WithReusePort(true) are enabled.
// Use Engine.Stop instead.
func Stop(ctx context.Context, protoAddr string) error {
	var eng *engine
	if s, ok := allEngines.Load(protoAddr); ok {
		eng = s.(*engine)
		eng.shutdown(nil)
		defer allEngines.Delete(protoAddr)
	} else {
		return errors.ErrEngineInShutdown
	}

	if eng.isShutdown() {
		return errors.ErrEngineInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func parseProtoAddr(protoAddr string) (string, string, error) {
	protoAddr = strings.ToLower(protoAddr)
	if strings.Count(protoAddr, "://") != 1 {
		return "", "", errors.ErrInvalidNetworkAddress
	}
	pair := strings.SplitN(protoAddr, "://", 2)
	proto, addr := pair[0], pair[1]
	switch proto {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "unix":
	default:
		return "", "", errors.ErrUnsupportedProtocol
	}
	if addr == "" {
		return "", "", errors.ErrInvalidNetworkAddress
	}
	return proto, addr, nil
}
