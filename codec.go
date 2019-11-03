package gnet

import (
	"bytes"
)

var CRLFByte = byte('\n')

type ICodec interface {
	Encode(buf []byte) ([]byte, error)
	Decode(c Conn) ([]byte, error)
}

//type Decoder interface {
//	Decode(c Conn) ([]byte, error)
//}
//type Encoder interface {
//	Encode(buf []byte) ([]byte, error)
//}

type BuiltInFrameCodec struct {
}

func (cc *BuiltInFrameCodec) Encode(buf []byte) ([]byte, error) {
	return buf, nil
}

func (cc *BuiltInFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	c.ResetBuffer()
	return buf, nil
}

type LineBasedFrameCodec struct {
}

func (cc *LineBasedFrameCodec) Encode(buf []byte) ([]byte, error) {
	return append(buf, CRLFByte), nil
}

func (cc *LineBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, CRLFByte)
	if idx == -1 {
		return nil, ErrCRLFNotFound
	}
	_, buf = c.ReadN(idx + 1)
	return buf[:idx], nil
}

type DelimiterBasedFrameCodec struct {
	Delimiter byte
}

func (cc *DelimiterBasedFrameCodec) Encode(buf []byte) ([]byte, error) {
	return append(buf, cc.Delimiter), nil
}

func (cc *DelimiterBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, cc.Delimiter)
	if idx == -1 {
		return nil, ErrDelimiterNotFound
	}
	_, buf = c.ReadN(idx + 1)
	return buf[:idx], nil
}

type FixedLengthFrameCodec struct {
	FrameLength int
}

func (cc *FixedLengthFrameCodec) Encode(buf []byte) ([]byte, error) {
	if len(buf)%cc.FrameLength != 0 {
		return nil, ErrInvalidFixedLength
	}
	return buf, nil
}

func (cc *FixedLengthFrameCodec) Decode(c Conn) ([]byte, error) {
	size, buf := c.ReadN(cc.FrameLength)
	if size == 0 {
		return nil, ErrUnexpectedEOF
	}
	return buf, nil
}

type LengthFieldBasedFrameCodec struct {
}
