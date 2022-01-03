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

//go:build !linux && !freebsd && !dragonfly && !darwin
// +build !linux,!freebsd,!dragonfly,!darwin

package gnet

import (
	"github.com/panjf2000/gnet/v2/pkg/errors"
)

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
