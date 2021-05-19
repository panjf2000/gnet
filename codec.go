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

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	errorset "github.com/panjf2000/gnet/errors"
)

// CRLFByte represents a byte of CRLF.
var CRLFByte = byte('\n')

type (
	// ICodec is the interface of gnet codec.
	ICodec interface {
		// Encode encodes frames upon server responses into TCP stream.
		Encode(c Conn, buf []byte) ([]byte, error)
		// Decode decodes frames from TCP stream via specific implementation.
		Decode(c Conn) ([]byte, error)
	}

	// BuiltInFrameCodec is the built-in codec which will be assigned to gnet server when customized codec is not set up.
	BuiltInFrameCodec struct{}

	// LineBasedFrameCodec encodes/decodes line-separated frames into/from TCP stream.
	LineBasedFrameCodec struct{}

	// DelimiterBasedFrameCodec encodes/decodes specific-delimiter-separated frames into/from TCP stream.
	DelimiterBasedFrameCodec struct {
		delimiter byte
	}

	// FixedLengthFrameCodec encodes/decodes fixed-length-separated frames into/from TCP stream.
	FixedLengthFrameCodec struct {
		frameLength int
	}

	// LengthFieldBasedFrameCodec is the refactoring from
	// https://github.com/smallnest/goframe/blob/master/length_field_based_frameconn.go, licensed by Apache License 2.0.
	// It encodes/decodes frames into/from TCP stream with value of the length field in the message.
	LengthFieldBasedFrameCodec struct {
		encoderConfig EncoderConfig
		decoderConfig DecoderConfig
	}
)

// Encode ...
func (cc *BuiltInFrameCodec) Encode(c Conn, buf []byte) ([]byte, error) {
	return buf, nil
}

// Decode ...
func (cc *BuiltInFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	if len(buf) == 0 {
		return nil, nil
	}
	c.ResetBuffer()
	return buf, nil
}

// Encode ...
func (cc *LineBasedFrameCodec) Encode(c Conn, buf []byte) ([]byte, error) {
	return append(buf, CRLFByte), nil
}

// Decode ...
func (cc *LineBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, CRLFByte)
	if idx == -1 {
		return nil, errorset.ErrCRLFNotFound
	}
	c.ShiftN(idx + 1)
	return buf[:idx], nil
}

// NewDelimiterBasedFrameCodec instantiates and returns a codec with a specific delimiter.
func NewDelimiterBasedFrameCodec(delimiter byte) *DelimiterBasedFrameCodec {
	return &DelimiterBasedFrameCodec{delimiter}
}

// Encode ...
func (cc *DelimiterBasedFrameCodec) Encode(c Conn, buf []byte) ([]byte, error) {
	return append(buf, cc.delimiter), nil
}

// Decode ...
func (cc *DelimiterBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	buf := c.Read()
	idx := bytes.IndexByte(buf, cc.delimiter)
	if idx == -1 {
		return nil, errorset.ErrDelimiterNotFound
	}
	c.ShiftN(idx + 1)
	return buf[:idx], nil
}

// NewFixedLengthFrameCodec instantiates and returns a codec with fixed length.
func NewFixedLengthFrameCodec(frameLength int) *FixedLengthFrameCodec {
	return &FixedLengthFrameCodec{frameLength}
}

// Encode ...
func (cc *FixedLengthFrameCodec) Encode(c Conn, buf []byte) ([]byte, error) {
	if len(buf)%cc.frameLength != 0 {
		return nil, errorset.ErrInvalidFixedLength
	}
	return buf, nil
}

// Decode ...
func (cc *FixedLengthFrameCodec) Decode(c Conn) ([]byte, error) {
	size, buf := c.ReadN(cc.frameLength)
	if size == 0 {
		return nil, errorset.ErrUnexpectedEOF
	}
	c.ShiftN(size)
	return buf, nil
}

// NewLengthFieldBasedFrameCodec instantiates and returns a codec based on the length field.
// It is the go implementation of netty LengthFieldBasedFrameecoder and LengthFieldPrepender.
// you can see javadoc of them to learn more details.
func NewLengthFieldBasedFrameCodec(ec EncoderConfig, dc DecoderConfig) *LengthFieldBasedFrameCodec {
	return &LengthFieldBasedFrameCodec{encoderConfig: ec, decoderConfig: dc}
}

// EncoderConfig config for encoder.
type EncoderConfig struct {
	// ByteOrder is the ByteOrder of the length field.
	ByteOrder binary.ByteOrder
	// LengthFieldLength is the length of the length field.
	LengthFieldLength int
	// LengthAdjustment is the compensation value to add to the value of the length field
	LengthAdjustment int
	// LengthIncludesLengthFieldLength is true, the length of the prepended length field is added to the value of
	// the prepended length field
	LengthIncludesLengthFieldLength bool
}

// DecoderConfig config for decoder.
type DecoderConfig struct {
	// ByteOrder is the ByteOrder of the length field.
	ByteOrder binary.ByteOrder
	// LengthFieldOffset is the offset of the length field
	LengthFieldOffset int
	// LengthFieldLength is the length of the length field
	LengthFieldLength int
	// LengthAdjustment is the compensation value to add to the value of the length field
	LengthAdjustment int
	// InitialBytesToStrip is the number of first bytes to strip out from the decoded frame
	InitialBytesToStrip int
}

// Encode ...
func (cc *LengthFieldBasedFrameCodec) Encode(c Conn, buf []byte) (out []byte, err error) {
	length := len(buf) + cc.encoderConfig.LengthAdjustment
	if cc.encoderConfig.LengthIncludesLengthFieldLength {
		length += cc.encoderConfig.LengthFieldLength
	}

	if length < 0 {
		return nil, errorset.ErrTooLessLength
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
		return nil, errorset.ErrUnsupportedLength
	}

	out = append(out, buf...)
	return
}

type innerBuffer []byte

func (in *innerBuffer) readN(n int) (buf []byte, err error) {
	if n == 0 {
		return nil, nil
	}

	if n < 0 {
		return nil, errors.New("negative length is invalid")
	} else if n > len(*in) {
		return nil, errors.New("exceeding buffer length")
	}
	buf = (*in)[:n]
	*in = (*in)[n:]
	return
}

// Decode ...
func (cc *LengthFieldBasedFrameCodec) Decode(c Conn) ([]byte, error) {
	var (
		in     innerBuffer
		header []byte
		err    error
	)
	in = c.Read()
	if cc.decoderConfig.LengthFieldOffset > 0 { // discard header(offset)
		header, err = in.readN(cc.decoderConfig.LengthFieldOffset)
		if err != nil {
			return nil, errorset.ErrUnexpectedEOF
		}
	}

	lenBuf, frameLength, err := cc.getUnadjustedFrameLength(&in)
	if err != nil {
		return nil, err
	}

	// real message length
	msgLength := int(frameLength) + cc.decoderConfig.LengthAdjustment
	msg, err := in.readN(msgLength)
	if err != nil {
		return nil, errorset.ErrUnexpectedEOF
	}

	fullMessage := make([]byte, len(header)+len(lenBuf)+msgLength)
	copy(fullMessage, header)
	copy(fullMessage[len(header):], lenBuf)
	copy(fullMessage[len(header)+len(lenBuf):], msg)
	c.ShiftN(len(fullMessage))
	return fullMessage[cc.decoderConfig.InitialBytesToStrip:], nil
}

func (cc *LengthFieldBasedFrameCodec) getUnadjustedFrameLength(in *innerBuffer) ([]byte, uint64, error) {
	switch cc.decoderConfig.LengthFieldLength {
	case 1:
		b, err := in.readN(1)
		if err != nil {
			return nil, 0, errorset.ErrUnexpectedEOF
		}
		return b, uint64(b[0]), nil
	case 2:
		lenBuf, err := in.readN(2)
		if err != nil {
			return nil, 0, errorset.ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint16(lenBuf)), nil
	case 3:
		lenBuf, err := in.readN(3)
		if err != nil {
			return nil, 0, errorset.ErrUnexpectedEOF
		}
		return lenBuf, readUint24(cc.decoderConfig.ByteOrder, lenBuf), nil
	case 4:
		lenBuf, err := in.readN(4)
		if err != nil {
			return nil, 0, errorset.ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint32(lenBuf)), nil
	case 8:
		lenBuf, err := in.readN(8)
		if err != nil {
			return nil, 0, errorset.ErrUnexpectedEOF
		}
		return lenBuf, cc.decoderConfig.ByteOrder.Uint64(lenBuf), nil
	default:
		return nil, 0, errorset.ErrUnsupportedLength
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
