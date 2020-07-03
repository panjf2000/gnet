// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build !linux,!freebsd,!dragonfly,!darwin,!windows

package gnet

type server struct {
	subEventLoopSet loadBalancer // event-loops for handling events
}

type eventloop struct {
	connCount int32 // number of active connections in event-loop
}

type listener struct {
	reusePort     bool
	addr, network string
}

func (ln *listener) normalize() error {
	return nil
}

func (ln *listener) close() {
}

func initListener(network, addr string, reusePort bool) (l *listener, err error) {
	l = &listener{network: network, addr: addr, reusePort: reusePort}
	err = l.normalize()
	return
}

func serve(_ EventHandler, _ *listener, _ *Options) error {
	return ErrUnsupportedPlatform
}
