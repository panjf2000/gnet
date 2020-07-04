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

// +build !linux,!freebsd,!dragonfly,!darwin,!windows

package gnet

import "github.com/panjf2000/gnet/errors"

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
	return errors.ErrUnsupportedPlatform
}
