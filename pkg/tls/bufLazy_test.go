package tls

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func eval(t *testing.T, buf *LazyBuffer, data []byte, n int, isLazy bool) {
	assert.EqualValues(t, buf.Bytes(), data)
	assert.EqualValues(t, buf.Len(), n)
	assert.EqualValues(t, buf.IsLazy(), isLazy)
}

func TestBufLazyEmpty(t *testing.T) {
	var buf LazyBuffer
	eval(t, &buf, []byte(nil), 0, true)
	buf.Write(nil)
	eval(t, &buf, []byte{}, 0, false)
}

func TestBufLazyLazyMode(t *testing.T) {
	var buf LazyBuffer
	var data []byte = []byte("Hello World!")
	buf.Set(data)
	eval(t, &buf, data, len(data), true)

	// Next 1 byte
	assert.EqualValues(t, buf.Next(1), data[:1])
	eval(t, &buf, data[1:], len(data)-1, true)

	// Next remaining byte
	assert.EqualValues(t, buf.Next(len(data)-1), data[1:])
	eval(t, &buf, make([]byte, 0), 0, true)

	// Next 1 more byte
	assert.EqualValues(t, buf.Next(1), data[:0])
	eval(t, &buf, make([]byte, 0), 0, true)

	// Reset the data, must be in lazy mode as the previous data is drained
	buf.Set(data)
	eval(t, &buf, data, len(data), true)

	// Next all byte + 1
	assert.EqualValues(t, buf.Next(len(data)+1), data)
	eval(t, &buf, make([]byte, 0), 0, true)

	// Done
	buf.Done()
	eval(t, &buf, []byte(nil), 0, true)
}

func TestBufLazyLazyToWriteMode(t *testing.T) {
	var buf LazyBuffer
	var data []byte = []byte("Hello World!")
	buf.Set(data)
	eval(t, &buf, data, len(data), true)

	// switch to write
	buf.Set(data)
	doubleData := append(data, data...)
	eval(t, &buf, doubleData, len(doubleData), false)

	// append new data
	doubleData = append(doubleData, data...)
	buf.Set(data)
	eval(t, &buf, doubleData, len(doubleData), false)

	buf.Done()
	eval(t, &buf, []byte(nil), 0, true)
}

func TestBufLazyWriteMode(t *testing.T) {
	var buf LazyBuffer
	var data []byte = []byte(strings.Repeat("A", defaultSize))

	buf.Grow(defaultSize)
	// fill up the default buffer
	buf.Write(data)
	eval(t, &buf, data, len(data), false)

	// grow the buffer
	buf.Write(data[:1])
	doubleData := append(data, data...)
	eval(t, &buf, doubleData[:len(data)+1], len(data)+1, false)

	// fill up the remaining buffer
	buf.Write(data[1:])
	eval(t, &buf, doubleData, len(doubleData), false)

	// consume half of the data
	assert.EqualValues(t, buf.Next(len(data)), data)
	eval(t, &buf, data, len(data), false)

	// consume 1 byte
	assert.EqualValues(t, buf.Next(1), data[:1])
	eval(t, &buf, data[1:], len(data)-1, false)

	// grow 1 byte, the data is copied to the beginning
	// slide things down
	buf.Grow(1)
	eval(t, &buf, data[1:], len(data)-1, false)

	// Done
	buf.Done()
	eval(t, &buf, []byte(nil), 0, true)
}
