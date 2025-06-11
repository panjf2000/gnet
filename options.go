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

package gnet

import (
	"time"

	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// Option is a function that will set up option.
type Option func(opts *Options)

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// TCPSocketOpt is the type of TCP socket options.
type TCPSocketOpt int

// Available TCP socket options.
const (
	TCPNoDelay TCPSocketOpt = iota
	TCPDelay
)

// Options are configurations for the gnet application.
type Options struct {
	// LB represents the load-balancing algorithm used when assigning new connections
	// to event loops. This option is server-only, and it is not applicable to the client.
	LB LoadBalancing

	// ReuseAddr indicates whether to set the SO_REUSEADDR socket option.
	// This option is server-only.
	ReuseAddr bool

	// ReusePort indicates whether to set the SO_REUSEPORT socket option.
	// This option is server-only.
	ReusePort bool

	// MulticastInterfaceIndex is the index of the interface name where the multicast UDP addresses will be bound to.
	// This option is server-only.
	MulticastInterfaceIndex int

	// BindToDevice is the name of the interface to which the listening socket will be bound.
	// It is only available on Linux at the moment, an error will therefore be returned when
	// setting this option on non-linux platforms.
	// This option is server-only.
	BindToDevice string

	// Multicore indicates whether the engine will be effectively created with multi-cores, if so,
	// then you must take care with synchronizing memory between all event callbacks; otherwise,
	// it will run the engine with single thread. The number of threads in the engine will be
	// automatically assigned to the number of usable logical CPUs that can be leveraged by the
	// current process.
	Multicore bool

	// NumEventLoop is set up to start the given number of event-loop goroutines.
	// Note that a non-negative NumEventLoop will override Multicore.
	NumEventLoop int

	// ReadBufferCap is the maximum number of bytes that can be read from the remote when the readable event comes.
	// The default value is 64KB, it can either be reduced to avoid starving the subsequent connections or increased
	// to read more data from a socket.
	//
	// Note that ReadBufferCap will always be converted to the least power of two integer value greater than
	// or equal to its real amount.
	ReadBufferCap int

	// WriteBufferCap is the maximum number of bytes that a static outbound buffer can hold,
	// if the data exceeds this value, the overflow bytes will be stored in the elastic linked list buffer.
	// The default value is 64KB.
	//
	// Note that WriteBufferCap will always be converted to the least power of two integer value greater than
	// or equal to its real amount.
	WriteBufferCap int

	// LockOSThread is used to determine whether each I/O event-loop should be associated to an OS thread,
	// it is useful when you need some kind of mechanisms like thread local storage, or invoke certain C
	// libraries (such as graphics lib: GLib) that require thread-level manipulation via cgo, or want all I/O
	// event-loops to actually run in parallel for a potential higher performance.
	LockOSThread bool

	// Ticker indicates whether the ticker has been set up.
	Ticker bool

	// TCPKeepAlive enables the TCP keep-alive mechanism (SO_KEEPALIVE) and set its value
	// on TCP_KEEPIDLE.
	// When TCPKeepInterval is not set, 1/5 of TCPKeepAlive will be set on TCP_KEEPINTVL,
	// and 5 will be set on TCP_KEEPCNT if TCPKeepCount is not assigned to a positive value.
	TCPKeepAlive time.Duration

	// TCPKeepInterval is the value for TCP_KEEPINTVL, it's the interval between
	// TCP keep-alive probes.
	TCPKeepInterval time.Duration

	// TCPKeepCount is the number of keep-alive probes that will be sent before
	// the connection is considered dead and dropped.
	TCPKeepCount int

	// TCPNoDelay controls whether the operating system should delay
	// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
	// When this option is assigned to TCPNoDelay, TCP_NODELAY socket option will
	// be turned on, on the contrary, if it is assigned to TCPDelay, the socket
	// option will be turned off.
	//
	// The default is TCPNoDelay, meaning that TCP_NODELAY is turned on and data
	// will not be buffered but sent as soon as possible after a write operation.
	TCPNoDelay TCPSocketOpt

	// SocketRecvBuffer sets the maximum socket receive buffer of kernel in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer of kernel in bytes.
	SocketSendBuffer int

	// LogPath specifies a local path where logs will be written, this is the easiest
	// way to set up logging, gnet instantiates a default uber-go/zap logger with this
	// given log path, you are also allowed to employ your own logger during the lifetime
	// by implementing the following logging.Logger interface.
	//
	// Note that this option can be overridden by a non-nil option Logger.
	LogPath string

	// LogLevel specifies the logging level, it should be used along with LogPath.
	LogLevel logging.Level

	// Logger is the customized logger for logging info, if it is not set,
	// then gnet will use the default logger powered by go.uber.org/zap.
	Logger logging.Logger

	// EdgeTriggeredIO enables the edge-triggered I/O for the underlying epoll/kqueue event-loop.
	// Don't enable it unless you are 100% sure what you are doing.
	// Note that this option is only available for stream-oriented protocol.
	EdgeTriggeredIO bool

	// EdgeTriggeredIOChunk specifies the number of bytes that `gnet` can
	// read/write up to in one event loop of ET. This option implies
	// EdgeTriggeredIO when it is set to a value greater than 0.
	// If EdgeTriggeredIO is set to true and EdgeTriggeredIOChunk is not set,
	// 1MB is used. The value of EdgeTriggeredIOChunk must be a power of 2,
	// otherwise, it will be rounded up to the nearest power of 2.
	EdgeTriggeredIOChunk int
}

// WithOptions sets up all options.
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithMulticore enables multi-cores mode for gnet engine.
func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

// WithLockOSThread enables LockOSThread mode for I/O event-loops.
func WithLockOSThread(lockOSThread bool) Option {
	return func(opts *Options) {
		opts.LockOSThread = lockOSThread
	}
}

// WithReadBufferCap sets ReadBufferCap for reading bytes.
func WithReadBufferCap(readBufferCap int) Option {
	return func(opts *Options) {
		opts.ReadBufferCap = readBufferCap
	}
}

// WithWriteBufferCap sets WriteBufferCap for pending bytes.
func WithWriteBufferCap(writeBufferCap int) Option {
	return func(opts *Options) {
		opts.WriteBufferCap = writeBufferCap
	}
}

// WithLoadBalancing picks the load-balancing algorithm for gnet engine.
func WithLoadBalancing(lb LoadBalancing) Option {
	return func(opts *Options) {
		opts.LB = lb
	}
}

// WithNumEventLoop sets the number of event loops for gnet engine.
func WithNumEventLoop(numEventLoop int) Option {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

// WithReusePort sets SO_REUSEPORT socket option.
func WithReusePort(reusePort bool) Option {
	return func(opts *Options) {
		opts.ReusePort = reusePort
	}
}

// WithReuseAddr sets SO_REUSEADDR socket option.
func WithReuseAddr(reuseAddr bool) Option {
	return func(opts *Options) {
		opts.ReuseAddr = reuseAddr
	}
}

// WithTCPKeepAlive enables the TCP keep-alive mechanism and sets its values.
func WithTCPKeepAlive(tcpKeepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}

// WithTCPKeepInterval sets the interval between TCP keep-alive probes.
func WithTCPKeepInterval(tcpKeepInterval time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepInterval = tcpKeepInterval
	}
}

// WithTCPKeepCount sets the number of keep-alive probes that will be sent before
// the connection is considered dead and dropped.
func WithTCPKeepCount(tcpKeepCount int) Option {
	return func(opts *Options) {
		opts.TCPKeepCount = tcpKeepCount
	}
}

// WithTCPNoDelay enable/disable the TCP_NODELAY socket option.
func WithTCPNoDelay(tcpNoDelay TCPSocketOpt) Option {
	return func(opts *Options) {
		opts.TCPNoDelay = tcpNoDelay
	}
}

// WithSocketRecvBuffer sets the maximum socket receive buffer of kernel in bytes.
func WithSocketRecvBuffer(recvBuf int) Option {
	return func(opts *Options) {
		opts.SocketRecvBuffer = recvBuf
	}
}

// WithSocketSendBuffer sets the maximum socket send buffer of kernel in bytes.
func WithSocketSendBuffer(sendBuf int) Option {
	return func(opts *Options) {
		opts.SocketSendBuffer = sendBuf
	}
}

// WithTicker indicates whether a ticker is currently set.
func WithTicker(ticker bool) Option {
	return func(opts *Options) {
		opts.Ticker = ticker
	}
}

// WithLogPath specifies a local path for logging file.
func WithLogPath(fileName string) Option {
	return func(opts *Options) {
		opts.LogPath = fileName
	}
}

// WithLogLevel specifies the logging level for the local logging file.
func WithLogLevel(lvl logging.Level) Option {
	return func(opts *Options) {
		opts.LogLevel = lvl
	}
}

// WithLogger specifies a customized logger.
func WithLogger(logger logging.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithMulticastInterfaceIndex sets the interface name where UDP multicast sockets will be bound to.
func WithMulticastInterfaceIndex(idx int) Option {
	return func(opts *Options) {
		opts.MulticastInterfaceIndex = idx
	}
}

// WithBindToDevice sets the name of the interface to which the listening socket will be bound.
//
// It is only available on Linux at the moment, an error will therefore be returned when
// setting this option on non-linux platforms.
func WithBindToDevice(iface string) Option {
	return func(opts *Options) {
		opts.BindToDevice = iface
	}
}

// WithEdgeTriggeredIO enables the edge-triggered I/O for the underlying epoll/kqueue event-loop.
func WithEdgeTriggeredIO(et bool) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIO = et
	}
}

// WithEdgeTriggeredIOChunk sets the number of bytes that `gnet` can
// read/write up to in one event loop of ET.
func WithEdgeTriggeredIOChunk(chunk int) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIOChunk = chunk
	}
}
