package gnet

import "errors"

var (
	// errShutdown server is closing.
	errShutdown = errors.New("server is going to be shutdown")
	// ErrInvalidFixedLength invalid fixed length.
	ErrInvalidFixedLength = errors.New("invalid fixed length of bytes")
	// ErrUnexpectedEOF no enough data to read.
	ErrUnexpectedEOF = errors.New("there is no enough data")
	// ErrDelimiterNotFound no such a delimiter.
	ErrDelimiterNotFound = errors.New("there is no such a delimiter")
	// ErrCRLFNotFound CRLF not found.
	ErrCRLFNotFound = errors.New("there is no CRLF")
	// ErrUnsupportedLength unsupported lengthFieldLength.
	ErrUnsupportedLength = errors.New("unsupported lengthFieldLength. (expected: 1, 2, 3, 4, or 8)")
	// ErrTooLessLength adjusted frame length is less than zero.
	ErrTooLessLength = errors.New("adjusted frame length is less than zero")
)
