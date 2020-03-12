package gnet

import "errors"

var (
	// ErrProtocolNotSupported occurs when trying to use protocol that is not supported.
	ErrProtocolNotSupported = errors.New("not supported protocol on this platform")
	// ErrServerShutdown occurs when server is closing.
	ErrServerShutdown = errors.New("server is going to be shutdown")
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
)
