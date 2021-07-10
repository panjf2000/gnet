// Copyright (c) 2019 Andy Pan
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
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/panjf2000/gnet/logging"
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

// Options are set when the client opens.
type Options struct {
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care with synchronizing memory between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of logical CPUs usable by the current process.
	Multicore bool

	// LockOSThread is used to determine whether each I/O event-loop is associated to an OS thread, it is useful when you
	// need some kind of mechanisms like thread local storage, or invoke certain C libraries (such as graphics lib: GLib)
	// that require thread-level manipulation via cgo, or want all I/O event-loops to actually run in parallel for a
	// potential higher performance.
	LockOSThread bool

	// ReadBufferCap is the maximum number of bytes that can be read from the client when the readable event comes.
	// The default value is 64KB, it can be reduced to avoid starving subsequent client connections.
	//
	// Note that ReadBufferCap will be always converted to the least power of two integer value greater than
	// or equal to its real amount.
	ReadBufferCap int

	// LB represents the load-balancing algorithm used when assigning new connections.
	LB LoadBalancing

	// NumEventLoop is set up to start the given number of event-loop goroutine.
	// Note: Setting up NumEventLoop will override Multicore.
	NumEventLoop int

	// ReusePort indicates whether to set up the SO_REUSEPORT socket option.
	ReusePort bool

	// Ticker indicates whether the ticker has been set up.
	Ticker bool

	// TCPKeepAlive sets up a duration for (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration

	// TCPNoDelay controls whether the operating system should delay
	// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
	//
	// The default is true (no delay), meaning that data is sent
	// as soon as possible after a Write.
	TCPNoDelay TCPSocketOpt

	// SocketRecvBuffer sets the maximum socket receive buffer in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer in bytes.
	SocketSendBuffer int

	// ICodec encodes and decodes TCP stream.
	Codec ICodec

	// LogPath the local path where logs will be written, this is the easiest way to set up client logs,
	// the client instantiates a default uber-go/zap logger with this given log path, you are also allowed to employ
	// you own logger during the client lifetime by implementing the following log.Logger interface.
	//
	// Note that this option can be overridden by the option Logger.
	LogPath string

	// LogLevel indicates the logging level inside client, it should be used along with LogPath.
	LogLevel zapcore.Level

	// Logger is the customized logger for logging info, if it is not set,
	// then gnet will use the default logger powered by go.uber.org/zap.
	Logger logging.Logger
}

// WithOptions sets up all options.
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithMulticore sets up multi-cores in gnet server.
func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

// WithLockOSThread sets up LockOSThread mode for I/O event-loops.
func WithLockOSThread(lockOSThread bool) Option {
	return func(opts *Options) {
		opts.LockOSThread = lockOSThread
	}
}

// WithReadBufferCap sets up ReadBufferCap for reading bytes.
func WithReadBufferCap(readBufferCap int) Option {
	return func(opts *Options) {
		opts.ReadBufferCap = readBufferCap
	}
}

// WithLoadBalancing sets up the load-balancing algorithm in gnet server.
func WithLoadBalancing(lb LoadBalancing) Option {
	return func(opts *Options) {
		opts.LB = lb
	}
}

// WithNumEventLoop sets up NumEventLoop in gnet server.
func WithNumEventLoop(numEventLoop int) Option {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

// WithReusePort sets up SO_REUSEPORT socket option.
func WithReusePort(reusePort bool) Option {
	return func(opts *Options) {
		opts.ReusePort = reusePort
	}
}

// WithTCPKeepAlive sets up the SO_KEEPALIVE socket option with duration.
func WithTCPKeepAlive(tcpKeepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}

// WithTCPNoDelay enable/disable the TCP_NODELAY socket option.
func WithTCPNoDelay(tcpNoDelay TCPSocketOpt) Option {
	return func(opts *Options) {
		opts.TCPNoDelay = tcpNoDelay
	}
}

// WithSocketRecvBuffer sets the maximum socket receive buffer in bytes.
func WithSocketRecvBuffer(recvBuf int) Option {
	return func(opts *Options) {
		opts.SocketRecvBuffer = recvBuf
	}
}

// WithSocketSendBuffer sets the maximum socket send buffer in bytes.
func WithSocketSendBuffer(sendBuf int) Option {
	return func(opts *Options) {
		opts.SocketSendBuffer = sendBuf
	}
}

// WithTicker indicates that a ticker is set.
func WithTicker(ticker bool) Option {
	return func(opts *Options) {
		opts.Ticker = ticker
	}
}

// WithCodec sets up a codec to handle TCP stream.
func WithCodec(codec ICodec) Option {
	return func(opts *Options) {
		opts.Codec = codec
	}
}

// WithLogPath is an option to set up the local path of log file.
func WithLogPath(fileName string) Option {
	return func(opts *Options) {
		opts.LogPath = fileName
	}
}

// WithLogLevel is an option to set up the logging level.
func WithLogLevel(lvl zapcore.Level) Option {
	return func(opts *Options) {
		opts.LogLevel = lvl
	}
}

// WithLogger sets up a customized logger.
func WithLogger(logger logging.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
