// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/smallnest/goframe"
)

// CRLFByte represents a byte of CRLF.
var CRLFByte = byte('\n')

// ICodec is the interface of gnet codec.
type ICodec interface {
	// Encode encodes frames upon server responses into TCP stream.
	Encode(buf []byte) ([]byte, error)
	// Decode decodes frames from TCP stream via specific implementation.
	Decode(c Conn) ([]byte, error)
}

//type Decoder interface {
//	Decode(c Conn) ([]byte, error)
//}
//type Encoder interface {
//	Encode(buf []byte) ([]byte, error)
//}

// BuiltInFrameCodec is the built-in codec which will be assigned to gnet server when customized codec is not set up.
type BuiltInFrameCodec struct {
}

// Encode ...
func (cc *BuiltInFrameCodec) Encode(buf []byte) ([]byte, error) {
	return buf, nil
}

// Decode ...
func (cc *BuiltInFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	c.ResetBuffer()
	return buf, nil
}

// LineBasedFrameCodec encodes/decodes line-separated frames into/from TCP stream.
type LineBasedFrameCodec struct {
}

// Encode ...
func (cc *LineBasedFrameCodec) Encode(buf []byte) ([]byte, error) {
	return append(buf, CRLFByte), nil
}

// Decode ...
func (cc *LineBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, CRLFByte)
	if idx == -1 {
		return nil, ErrCRLFNotFound
	}
	_, buf = c.ReadN(idx + 1)
	return buf[:idx], nil
}

// DelimiterBasedFrameCodec encodes/decodes specific-delimiter-separated frames into/from TCP stream.
type DelimiterBasedFrameCodec struct {
	delimiter byte
}

// NewDelimiterBasedFrameCodec instantiates and returns a codec with a specific delimiter.
func NewDelimiterBasedFrameCodec(delimiter byte) *DelimiterBasedFrameCodec {
	return &DelimiterBasedFrameCodec{delimiter}
}

// Encode ...
func (cc *DelimiterBasedFrameCodec) Encode(buf []byte) ([]byte, error) {
	return append(buf, cc.delimiter), nil
}

// Decode ...
func (cc *DelimiterBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, cc.delimiter)
	if idx == -1 {
		return nil, ErrDelimiterNotFound
	}
	_, buf = c.ReadN(idx + 1)
	return buf[:idx], nil
}

// FixedLengthFrameCodec encodes/decodes fixed-length-separated frames into/from TCP stream.
type FixedLengthFrameCodec struct {
	frameLength int
}

// NewFixedLengthFrameCodec instantiates and returns a codec with fixed length.
func NewFixedLengthFrameCodec(frameLength int) *FixedLengthFrameCodec {
	return &FixedLengthFrameCodec{frameLength}
}

// Encode ...
func (cc *FixedLengthFrameCodec) Encode(buf []byte) ([]byte, error) {
	if len(buf)%cc.frameLength != 0 {
		return nil, ErrInvalidFixedLength
	}
	return buf, nil
}

// Decode ...
func (cc *FixedLengthFrameCodec) Decode(c Conn) ([]byte, error) {
	size, buf := c.ReadN(cc.frameLength)
	if size == 0 {
		return nil, ErrUnexpectedEOF
	}
	return buf, nil
}

// LengthFieldBasedFrameCodec is the refactoring from https://github.com/smallnest/goframe/blob/master/length_field_based_frameconn.go, licensed by Apache License 2.0.
// It encodes/decodes frames into/from TCP stream with value of the length field in the message.
// Original implementation: https://github.com/netty/netty/blob/4.1/codec/src/main/java/io/netty/handler/codec/LengthFieldBasedFrameDecoder.java
type LengthFieldBasedFrameCodec struct {
	encoderConfig EncoderConfig
	decoderConfig DecoderConfig
}

// NewLengthFieldBasedFrameCodec instantiates and returns a codec based on the length field.
// It is the go implementation of netty LengthFieldBasedFrameecoder and LengthFieldPrepender.
// you can see javadoc of them to learn more details.
func NewLengthFieldBasedFrameCodec(encoderConfig EncoderConfig, decoderConfig DecoderConfig) *LengthFieldBasedFrameCodec {
	return &LengthFieldBasedFrameCodec{encoderConfig, decoderConfig}
}

// EncoderConfig config for encoder.
type EncoderConfig = goframe.EncoderConfig
//type EncoderConfig struct {
//	// ByteOrder is the ByteOrder of the length field.
//	ByteOrder binary.ByteOrder
//	// LengthFieldLength is the length of the length field.
//	LengthFieldLength int
//	// LengthAdjustment is the compensation value to add to the value of the length field
//	LengthAdjustment int
//	// LengthIncludesLengthFieldLength is true, the length of the prepended length field is added to the value of the prepended length field
//	LengthIncludesLengthFieldLength bool
//}

// DecoderConfig config for decoder.
type DecoderConfig = goframe.DecoderConfig
//type DecoderConfig struct {
//	// ByteOrder is the ByteOrder of the length field.
//	ByteOrder binary.ByteOrder
//	// LengthFieldOffset is the offset of the length field
//	LengthFieldOffset int
//	// LengthFieldLength is the length of the length field
//	LengthFieldLength int
//	// LengthAdjustment is the compensation value to add to the value of the length field
//	LengthAdjustment int
//	// InitialBytesToStrip is the number of first bytes to strip out from the decoded frame
//	InitialBytesToStrip int
//}

// Encode ...
func (cc *LengthFieldBasedFrameCodec) Encode(buf []byte) (out []byte, err error) {
	length := len(buf) + cc.encoderConfig.LengthAdjustment
	if cc.encoderConfig.LengthIncludesLengthFieldLength {
		length += cc.encoderConfig.LengthFieldLength
	}

	if length < 0 {
		return nil, ErrTooLessLength
	}

	switch cc.encoderConfig.LengthFieldLength {
	case 1:
		if length >= 256 {
			return nil, fmt.Errorf("length does not fit into a byte: %d", length)
		}
		out = []byte{byte(length)}
	case 2:
		if length >= 65536 {
			return nil, fmt.Errorf("length does not fit into a short integer: %d", length)
		}
		out = make([]byte, 2)
		cc.encoderConfig.ByteOrder.PutUint16(out, uint16(length))
	case 3:
		if length >= 16777216 {
			return nil, fmt.Errorf("length does not fit into a medium integer: %d", length)
		}
		out = writeUint24(cc.encoderConfig.ByteOrder, length)
	case 4:
		out = make([]byte, 4)
		cc.encoderConfig.ByteOrder.PutUint32(out, uint32(length))
	case 8:
		out = make([]byte, 8)
		cc.encoderConfig.ByteOrder.PutUint64(out, uint64(length))
	default:
		return nil, ErrUnsupportedLength
	}

	out = append(out, buf...)
	return
}

// Decode ...
func (cc *LengthFieldBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	var size int
	var header []byte
	if cc.decoderConfig.LengthFieldOffset > 0 { //discard header(offset)
		size, header = c.ReadN(cc.decoderConfig.LengthFieldOffset)
		if size == 0 {
			return nil, ErrUnexpectedEOF
		}
	}

	lenBuf, frameLength, err := cc.getUnadjustedFrameLength(c)
	if err != nil {
		return nil, err
	}

	if cc.decoderConfig.LengthAdjustment > 0 { //discard adjust header
		size, _ := c.ReadN(cc.decoderConfig.LengthAdjustment)
		if size == 0 {
			return nil, ErrUnexpectedEOF
		}
	}

	// real message length
	msgLength := int(frameLength) + cc.decoderConfig.LengthAdjustment
	size, msg := c.ReadN(msgLength)
	if size == 0 {
		return nil, ErrUnexpectedEOF
	}

	fullMessage := make([]byte, len(header)+len(lenBuf)+msgLength)
	copy(fullMessage, header)
	copy(fullMessage[len(header):], lenBuf)
	copy(fullMessage[len(header)+len(lenBuf):], msg)
	return fullMessage[cc.decoderConfig.InitialBytesToStrip:], nil
}

func (cc *LengthFieldBasedFrameCodec) getUnadjustedFrameLength(c Conn) (lenBuf []byte, n uint64, err error) {
	switch cc.decoderConfig.LengthFieldLength {
	case 1:
		size, b := c.ReadN(1)
		if size == 0 {
			err = ErrUnexpectedEOF
		}
		return b, uint64(b[0]), err
	case 2:
		size, lenBuf := c.ReadN(2)
		if size == 0 {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint16(lenBuf)), nil
	case 3:
		size, lenBuf := c.ReadN(3)
		if size == 0 {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, readUint24(cc.decoderConfig.ByteOrder, lenBuf), nil
	case 4:
		size, lenBuf := c.ReadN(4)
		if size == 0 {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint32(lenBuf)), nil
	case 8:
		size, lenBuf := c.ReadN(8)
		if size == 0 {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint64(lenBuf)), nil
	default:
		return nil, 0, ErrUnsupportedLength
	}
}

func readUint24(byteOrder binary.ByteOrder, b []byte) uint64 {
	_ = b[2]
	if byteOrder == binary.LittleEndian {
		return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16
	}
	return uint64(b[2]) | uint64(b[1])<<8 | uint64(b[0])<<16
}

func writeUint24(byteOrder binary.ByteOrder, v int) []byte {
	b := make([]byte, 3)
	if byteOrder == binary.LittleEndian {
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		b[2] = byte(v >> 16)
	} else {
		b[2] = byte(v)
		b[1] = byte(v >> 8)
		b[0] = byte(v >> 16)
	}
	return b
}
