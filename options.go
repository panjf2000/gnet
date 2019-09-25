// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import "time"

type Option func(opts *Options)

func initOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// Options are set when the client opens.
type Options struct {
	// Multicore indicates whether the server will be effectively created with multi-cores, if so,
	// then you must take care with synchonizing memory between all event callbacks, otherwise,
	// it will run the server with single thread. The number of threads in the server will be automatically
	// assigned to the value of runtime.NumCPU().
	Multicore bool
	//ReusePort
	ReusePort bool
	// TCPKeepAlive (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

func WithReusePort(reusePort bool) Option {
	return func(opts *Options) {
		opts.ReusePort = reusePort
	}
}

func WithTCPKeepAlive(tcpKeepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}
