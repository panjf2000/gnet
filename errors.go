package gnet

import "errors"

var (
	// errShutdown indicates this server is closing.
	errShutdown = errors.New("server is going to be shutdown")
	// ErrInvalidFixedLength ...
	ErrInvalidFixedLength = errors.New("invalid fixed length of bytes")
	// ErrUnexpectedEOF ...
	ErrUnexpectedEOF = errors.New("there is no enough data")
	// ErrDelimiterNotFound ...
	ErrDelimiterNotFound = errors.New("there is no such a delimiter")
	// ErrDelimiterNotFound ...
	ErrCRLFNotFound = errors.New("there is no CRLF")
)
