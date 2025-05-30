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

// Package errors defines common errors for gnet.
package errors

import "errors"

var (
	// ErrEmptyEngine occurs when trying to do something with an empty engine.
	ErrEmptyEngine = errors.New("gnet: the internal engine is empty")
	// ErrEngineShutdown occurs when server is closing.
	ErrEngineShutdown = errors.New("gnet: server is going to be shutdown")
	// ErrEngineInShutdown occurs when attempting to shut the server down more than once.
	ErrEngineInShutdown = errors.New("gnet: server is already in shutdown")
	// ErrAcceptSocket occurs when acceptor does not accept the new connection properly.
	ErrAcceptSocket = errors.New("gnet: accept a new connection error")
	// ErrTooManyEventLoopThreads occurs when attempting to set up more than 10,000 event-loop goroutines under LockOSThread mode.
	ErrTooManyEventLoopThreads = errors.New("gnet: too many event-loops under LockOSThread mode")
	// ErrUnsupportedProtocol occurs when trying to use protocol that is not supported.
	ErrUnsupportedProtocol = errors.New("gnet: only unix, tcp/tcp4/tcp6, udp/udp4/udp6 are supported")
	// ErrUnsupportedTCPProtocol occurs when trying to use an unsupported TCP protocol.
	ErrUnsupportedTCPProtocol = errors.New("gnet: only tcp/tcp4/tcp6 are supported")
	// ErrUnsupportedUDPProtocol occurs when trying to use an unsupported UDP protocol.
	ErrUnsupportedUDPProtocol = errors.New("gnet: only udp/udp4/udp6 are supported")
	// ErrUnsupportedUDSProtocol occurs when trying to use an unsupported Unix protocol.
	ErrUnsupportedUDSProtocol = errors.New("gnet: only unix is supported")
	// ErrUnsupportedOp occurs when calling some methods that are either not supported or have not been implemented yet.
	ErrUnsupportedOp = errors.New("gnet: unsupported operation")
	// ErrNegativeSize occurs when trying to pass a negative size to a buffer.
	ErrNegativeSize = errors.New("gnet: negative size is not allowed")
	// ErrNoIPv4AddressOnInterface occurs when an IPv4 multicast address is set on an interface but IPv4 is not configured.
	ErrNoIPv4AddressOnInterface = errors.New("gnet: no IPv4 address on interface")
	// ErrInvalidNetworkAddress occurs when the network address is invalid.
	ErrInvalidNetworkAddress = errors.New("gnet: invalid network address")
	// ErrInvalidNetConn occurs when trying to do something with an empty net.Conn.
	ErrInvalidNetConn = errors.New("gnet: the net.Conn is empty")
	// ErrNilRunnable occurs when trying to execute a nil runnable.
	ErrNilRunnable = errors.New("gnet: nil runnable is not allowed")
)
