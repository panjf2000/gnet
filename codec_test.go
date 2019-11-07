package gnet

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func TestLengthFieldBasedFrameCodec(t *testing.T) {
	encoderConfig := EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               3,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoderConfig := DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   3,
		LengthAdjustment:    0,
		InitialBytesToStrip: 3,
	}
	codec := NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz := rand.Intn(10) * 64
	data := make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ := codec.Encode(data)
	if string(out[3:]) != string(data) {
		t.Fatalf("data don't match with big endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	encoderConfig.ByteOrder = binary.LittleEndian
	decoderConfig.ByteOrder = binary.LittleEndian
	codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz = rand.Intn(10) * 64
	data = make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ = codec.Encode(data)
	if string(out[3:]) != string(data) {
		t.Fatalf("data don't match with little endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	buf := make([]byte, 3)
	rand.Read(buf)
	bNum := readUint24(binary.BigEndian, buf)
	p := writeUint24(binary.BigEndian, int(bNum))
	if string(buf) != string(p) {
		t.Fatalf("data don't match, raw data: %s, recovered data: %s\n", string(buf), string(p))
	}
}
