package gnet

import (
	"bytes"
)

var CRLFByte = byte('\n')

type ICodec interface {
	Encode(buf []byte, c Conn) error
	Decode(c Conn) ([]byte, error)
}

//type Decoder interface {
//	Decode(c Conn) ([]byte, error)
//}
//type Encoder interface {
//	Encode(buf []byte) ([]byte, error)
//}

type LineBasedFrameCodec struct {
}

func (cc *LineBasedFrameCodec) Encode(buf []byte, c Conn) error {
	_, _ = c.OutboundBuffer().Write(buf)
	return c.OutboundBuffer().WriteByte(CRLFByte)
}

func (cc *LineBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, CRLFByte)
	if idx == -1 {
		return nil, ErrCRLFNotFound
	}
	return buf[:idx], nil
}

type DelimiterBasedFrameCodec struct {
	Delimiter byte
}

func (cc *DelimiterBasedFrameCodec) Encode(buf []byte, c Conn) error {
	_, _ = c.OutboundBuffer().Write(buf)
	return c.OutboundBuffer().WriteByte(cc.Delimiter)
}

func (cc *DelimiterBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.LastIndexByte(buf, cc.Delimiter)
	if idx == -1 {
		return nil, ErrDelimiterNotFound
	}
	return buf[:idx+1], nil
}

type FixedLengthFrameCodec struct {
	Step int
}

func (cc *FixedLengthFrameCodec) Encode(buf []byte, c Conn) ([]byte, error) {
	if len(buf)%cc.Step != 0 {
		return nil, ErrInvalidFixedLength
	}
	return buf, nil
}

func (cc *FixedLengthFrameCodec) Decode(c Conn) ([]byte, error) {
	size, buf := c.ReadN(cc.Step)
	if size == 0 {
		return nil, ErrUnexpectedEOF
	}
	return buf, nil
}

type LengthFieldBasedFrameCodec struct {
}
