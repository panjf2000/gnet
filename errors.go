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

import "errors"

var (
	// ErrUnsupportedProtocol occurs when trying to use protocol that is not supported.
	ErrUnsupportedProtocol = errors.New("unsupported protocol on this platform")
	// ErrUnsupportedPlatform occurs when running gnet on an unsupported platform.
	ErrUnsupportedPlatform = errors.New("unsupported platform in gnet")

	// errServerShutdown occurs when server is closing.
	errServerShutdown = errors.New("server is going to be shutdown")
	// errInvalidFixedLength occurs when the output data have invalid fixed length.
	errInvalidFixedLength = errors.New("invalid fixed length of bytes")
	// errUnexpectedEOF occurs when no enough data to read by codec.
	errUnexpectedEOF = errors.New("there is no enough data")
	// errDelimiterNotFound occurs when no such a delimiter is in input data.
	errDelimiterNotFound = errors.New("there is no such a delimiter")
	// errCRLFNotFound occurs when a CRLF is not found by codec.
	errCRLFNotFound = errors.New("there is no CRLF")
	// errUnsupportedLength occurs when unsupported lengthFieldLength is from input data.
	errUnsupportedLength = errors.New("unsupported lengthFieldLength. (expected: 1, 2, 3, 4, or 8)")
	// errTooLessLength occurs when adjusted frame length is less than zero.
	errTooLessLength = errors.New("adjusted frame length is less than zero")
)
