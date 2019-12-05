package gnet

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func TestLengthFieldBasedFrameCodecWith1(t *testing.T) {
	encoderConfig := EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               1,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoderConfig := DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   1,
		LengthAdjustment:    0,
		InitialBytesToStrip: 1,
	}
	codec := NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz := 255
	data := make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ := codec.Encode(nil, data)
	if string(out[1:]) != string(data) {
		t.Fatalf("data don't match with big endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	encoderConfig.ByteOrder = binary.LittleEndian
	decoderConfig.ByteOrder = binary.LittleEndian
	codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ = codec.Encode(nil, data)
	if string(out[1:]) != string(data) {
		t.Fatalf("data don't match with little endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}
}

func TestLengthFieldBasedFrameCodecWith2(t *testing.T) {
	encoderConfig := EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               2,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoderConfig := DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   2,
		LengthAdjustment:    0,
		InitialBytesToStrip: 2,
	}
	codec := NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz := rand.Intn(10) * 1024
	data := make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ := codec.Encode(nil, data)
	if string(out[2:]) != string(data) {
		t.Fatalf("data don't match with big endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	encoderConfig.ByteOrder = binary.LittleEndian
	decoderConfig.ByteOrder = binary.LittleEndian
	codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz = rand.Intn(10) * 1024
	data = make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ = codec.Encode(nil, data)
	if string(out[2:]) != string(data) {
		t.Fatalf("data don't match with little endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}
}

func TestLengthFieldBasedFrameCodecWith3(t *testing.T) {
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
	sz := rand.Intn(10) * 64 * 1024
	data := make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ := codec.Encode(nil, data)
	if string(out[3:]) != string(data) {
		t.Fatalf("data don't match with big endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	encoderConfig.ByteOrder = binary.LittleEndian
	decoderConfig.ByteOrder = binary.LittleEndian
	codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz = rand.Intn(10) * 64 * 1024
	data = make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ = codec.Encode(nil, data)
	if string(out[3:]) != string(data) {
		t.Fatalf("data don't match with little endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	buf := make([]byte, 3)
	rand.Read(buf)
	bNum := readUint24(binary.BigEndian, buf)
	p := writeUint24(binary.BigEndian, int(bNum))
	if string(buf) != string(p) {
		t.Fatalf("data don't match with big endian, raw data: %s, recovered data: %s\n", string(buf), string(p))
	}

	rand.Read(buf)
	bNum = readUint24(binary.LittleEndian, buf)
	p = writeUint24(binary.LittleEndian, int(bNum))
	if string(buf) != string(p) {
		t.Fatalf("data don't match with little endian, raw data: %s, recovered data: %s\n", string(buf), string(p))
	}
}

func TestLengthFieldBasedFrameCodecWith8(t *testing.T) {
	encoderConfig := EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               8,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoderConfig := DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   8,
		LengthAdjustment:    0,
		InitialBytesToStrip: 8,
	}
	codec := NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz := rand.Intn(10) * 1024 * 1024
	data := make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ := codec.Encode(nil, data)
	if string(out[8:]) != string(data) {
		t.Fatalf("data don't match with big endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}

	encoderConfig.ByteOrder = binary.LittleEndian
	decoderConfig.ByteOrder = binary.LittleEndian
	codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	sz = rand.Intn(10) * 1024 * 1024
	data = make([]byte, sz)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	out, _ = codec.Encode(nil, data)
	if string(out[8:]) != string(data) {
		t.Fatalf("data don't match with little endian, raw data: %s, encoded data: %s\n", string(data), string(out))
	}
}
