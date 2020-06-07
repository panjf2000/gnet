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
