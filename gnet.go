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
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2/internal/toolkit"
	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
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

// CountConnections counts the number of currently active connections and returns it.
func (s Engine) CountConnections() (count int) {
	s.eng.lb.iterate(func(i int, el *eventloop) bool {
		count += int(el.loadConn())
		return true
	})
	return
}

// Dup returns a copy of the underlying file descriptor of listener.
// It is the caller's responsibility to close dupFD when finished.
// Closing listener does not affect dupFD, and closing dupFD does not affect listener.
func (s Engine) Dup() (dupFD int, err error) {
	var sc string
	dupFD, sc, err = s.eng.ln.dup()
	if err != nil {
		logging.Warnf("%s failed when duplicating new fd\n", sc)
	}
	return
}

// Reader is an interface that consists of a number of methods for reading that Conn must implement.
type Reader interface {
	// ================================== Non-concurrency-safe API's ==================================

	io.Reader
	io.WriterTo // must be non-blocking, otherwise it may block the event-loop.

	// Next returns a slice containing the next n bytes from the buffer,
	// advancing the buffer as if the bytes had been returned by Read.
	// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
	// The error is ErrBufferFull if n is larger than b's buffer size.
	//
	// Note that the []byte buf returned by Next() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Next(n int) (buf []byte, err error)

	// Peek returns the next n bytes without advancing the reader. The bytes stop
	// being valid at the next read call. If Peek returns fewer than n bytes, it
	// also returns an error explaining why the read is short. The error is
	// ErrBufferFull if n is larger than b's buffer size.
	//
	// Note that the []byte buf returned by Peek() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Peek(n int) (buf []byte, err error)

	// Discard skips the next n bytes, returning the number of bytes discarded.
	//
	// If Discard skips fewer than n bytes, it also returns an error.
	// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
	// reading from the underlying io.Reader.
	Discard(n int) (discarded int, err error)

	// InboundBuffered returns the number of bytes that can be read from the current buffer.
	InboundBuffered() (n int)
}

// Writer is an interface that consists of a number of methods for writing that Conn must implement.
type Writer interface {
	// ================================== Non-concurrency-safe API's ==================================

	io.Writer
	io.ReaderFrom // must be non-blocking, otherwise it may block the event-loop.

	// Writev writes multiple byte slices to peer synchronously, you must call it in the current goroutine.
	Writev(bs [][]byte) (n int, err error)

	// Flush writes any buffered data to the underlying connection, you must call it in the current goroutine.
	Flush() (err error)

	// OutboundBuffered returns the number of bytes that can be read from the current buffer.
	OutboundBuffered() (n int)

	// ==================================== Concurrency-safe API's ====================================

	// AsyncWrite writes one byte slice to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWrite(buf []byte, callback AsyncCallback) (err error)

	// AsyncWritev writes multiple byte slices to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWritev(bs [][]byte, callback AsyncCallback) (err error)
}

// AsyncCallback is a callback which will be invoked after the asynchronous functions has finished executing.
//
// Note that the parameter gnet.Conn is already released under UDP protocol, thus it's not allowed to be accessed.
type AsyncCallback func(c Conn) error

// Socket is a set of functions which manipulate the underlying file descriptor of a connection.
type Socket interface {
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
	// CloseRead() error
	// CloseWrite() error
}

// Conn is an interface of underlying connection.
type Conn interface {
	Reader
	Writer
	Socket

	// ================================== Non-concurrency-safe API's ==================================

	// Context returns a user-defined context.
	Context() (ctx interface{})

	// SetContext sets a user-defined context.
	SetContext(ctx interface{})

	// LocalAddr is the connection's local socket address.
	LocalAddr() (addr net.Addr)

	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() (addr net.Addr)

	// SetDeadline implements net.Conn.
	SetDeadline(t time.Time) (err error)

	// SetReadDeadline implements net.Conn.
	SetReadDeadline(t time.Time) (err error)

	// SetWriteDeadline implements net.Conn.
	SetWriteDeadline(t time.Time) (err error)

	// ==================================== Concurrency-safe API's ====================================

	// Wake triggers a OnTraffic event for the connection.
	Wake(callback AsyncCallback) (err error)

	// CloseWithCallback closes the current connection, usually you don't need to pass a non-nil callback
	// because you should use OnClose() instead, the callback here is only for compatibility.
	CloseWithCallback(callback AsyncCallback) (err error)

	// Close closes the current connection, implements net.Conn.
	Close() (err error)
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
		// The parameter out is the return value which is going to be sent back to the peer.
		// Sending large amounts of data back to the peer in OnOpen is usually not recommended.
		OnOpen(c Conn) (out []byte, action Action)

		// OnClose fires when a connection has been closed.
		// The parameter err is the last known connection error.
		OnClose(c Conn, err error) (action Action)

		// OnTraffic fires when a socket receives data from the peer.
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
func (es *BuiltinEventEngine) OnBoot(_ Engine) (action Action) {
	return
}

// OnShutdown fires when the engine is being shut down, it is called right after
// all event-loops and connections are closed.
func (es *BuiltinEventEngine) OnShutdown(_ Engine) {
}

// OnOpen fires when a new connection has been opened.
// The parameter out is the return value which is going to be sent back to the peer.
func (es *BuiltinEventEngine) OnOpen(_ Conn) (out []byte, action Action) {
	return
}

// OnClose fires when a connection has been closed.
// The parameter err is the last known connection error.
func (es *BuiltinEventEngine) OnClose(_ Conn, _ error) (action Action) {
	return
}

// OnTraffic fires when a local socket receives data from the peer.
func (es *BuiltinEventEngine) OnTraffic(_ Conn) (action Action) {
	return
}

// OnTick fires immediately after the engine starts and will fire again
// following the duration specified by the delay return value.
func (es *BuiltinEventEngine) OnTick() (delay time.Duration, action Action) {
	return
}

// MaxStreamBufferCap is the default buffer size for each stream-oriented connection(TCP/Unix).
var MaxStreamBufferCap = 64 * 1024 // 64KB

// Run starts handling events on the specified address.
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
func Run(eventHandler EventHandler, protoAddr string, opts ...Option) (err error) {
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
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= ring.DefaultBufferSize:
		options.ReadBufferCap = ring.DefaultBufferSize
	default:
		options.ReadBufferCap = toolkit.CeilToPowerOfTwo(rbc)
	}
	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= ring.DefaultBufferSize:
		options.WriteBufferCap = ring.DefaultBufferSize
	default:
		options.WriteBufferCap = toolkit.CeilToPowerOfTwo(wbc)
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
	allEngines sync.Map

	// shutdownPollInterval is how often we poll to check whether engine has been shut down during gnet.Stop().
	shutdownPollInterval = 500 * time.Millisecond
)

// Stop gracefully shuts down the engine without interrupting any active event-loops,
// it waits indefinitely for connections and event-loops to be closed and then shuts down.
func Stop(ctx context.Context, protoAddr string) error {
	var eng *engine
	if s, ok := allEngines.Load(protoAddr); ok {
		eng = s.(*engine)
		eng.signalShutdown()
		defer allEngines.Delete(protoAddr)
	} else {
		return errors.ErrEngineInShutdown
	}

	if eng.isInShutdown() {
		return errors.ErrEngineInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if eng.isInShutdown() {
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

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}
