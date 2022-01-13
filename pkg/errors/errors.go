// Copyright (c) 2019 Andy Pan
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

package errors

import "errors"

var (
	// ErrEngineShutdown occurs when server is closing.
	ErrEngineShutdown = errors.New("server is going to be shutdown")
	// ErrEngineInShutdown occurs when attempting to shut the server down more than once.
	ErrEngineInShutdown = errors.New("server is already in shutdown")
	// ErrAcceptSocket occurs when acceptor does not accept the new connection properly.
	ErrAcceptSocket = errors.New("accept a new connection error")
	// ErrTooManyEventLoopThreads occurs when attempting to set up more than 10,000 event-loop goroutines under LockOSThread mode.
	ErrTooManyEventLoopThreads = errors.New("too many event-loops under LockOSThread mode")
	// ErrUnsupportedProtocol occurs when trying to use protocol that is not supported.
	ErrUnsupportedProtocol = errors.New("only unix, tcp/tcp4/tcp6, udp/udp4/udp6 are supported")
	// ErrUnsupportedTCPProtocol occurs when trying to use an unsupported TCP protocol.
	ErrUnsupportedTCPProtocol = errors.New("only tcp/tcp4/tcp6 are supported")
	// ErrUnsupportedUDPProtocol occurs when trying to use an unsupported UDP protocol.
	ErrUnsupportedUDPProtocol = errors.New("only udp/udp4/udp6 are supported")
	// ErrUnsupportedUDSProtocol occurs when trying to use an unsupported Unix protocol.
	ErrUnsupportedUDSProtocol = errors.New("only unix is supported")
	// ErrUnsupportedPlatform occurs when running gnet on an unsupported platform.
	ErrUnsupportedPlatform = errors.New("unsupported platform in gnet")
	// ErrUnsupportedOp occurs when calling some methods that has not been implemented yet.
	ErrUnsupportedOp = errors.New("unsupported operation")
	// ErrNegativeSize occurs when trying to pass a negative size to a buffer.
	ErrNegativeSize = errors.New("negative size is invalid")
)
