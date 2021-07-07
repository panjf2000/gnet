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

package errors

import "errors"

var (
	// ErrServerShutdown occurs when server is closing.
	ErrServerShutdown = errors.New("server is going to be shutdown")
	// ErrServerInShutdown occurs when attempting to shut the server down more than once.
	ErrServerInShutdown = errors.New("server is already in shutdown")
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

	// ================================================= codec errors =================================================.

	// ErrInvalidFixedLength occurs when the output data have invalid fixed length.
	ErrInvalidFixedLength = errors.New("invalid fixed length of bytes")
	// ErrUnexpectedEOF occurs when no enough data to read by codec.
	ErrUnexpectedEOF = errors.New("there is no enough data")
	// ErrDelimiterNotFound occurs when no such a delimiter is in input data.
	ErrDelimiterNotFound = errors.New("there is no such a delimiter")
	// ErrCRLFNotFound occurs when a CRLF is not found by codec.
	ErrCRLFNotFound = errors.New("there is no CRLF")
	// ErrUnsupportedLength occurs when unsupported lengthFieldLength is from input data.
	ErrUnsupportedLength = errors.New("unsupported lengthFieldLength. (expected: 1, 2, 3, 4, or 8)")
	// ErrTooLessLength occurs when adjusted frame length is less than zero.
	ErrTooLessLength = errors.New("adjusted frame length is less than zero")

	// =============================================== internal errors ===============================================.

	// ErrShortWritev occurs when internal/io.Writev fails to send all data.
	ErrShortWritev = errors.New("short writev")
	// ErrShortReadv occurs when internal/io.Readv fails to send all data.
	ErrShortReadv = errors.New("short readv")
)
