package elastic

import (
	"bytes"
	crand "crypto/rand"
	"math/rand"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMixedBuffer_Basic(t *testing.T) {
	const maxStaticSize = 4 * 1024
	mb, _ := New(maxStaticSize)
	const dataLen = 5 * 1024
	data := make([]byte, dataLen)
	_, err := crand.Read(data)
	require.NoError(t, err)
	n, err := mb.Write(data)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, n)
	require.EqualValues(t, dataLen, mb.Buffered())
	require.EqualValues(t, dataLen, mb.ringBuffer.Buffered())

	rbn := mb.ringBuffer.Len()
	mb.Reset(-1)
	newDataLen := rbn + 2*1024
	data = make([]byte, newDataLen)
	_, err = crand.Read(data)
	require.NoError(t, err)
	n, err = mb.Write(data)
	require.NoError(t, err)
	require.EqualValues(t, newDataLen, n)
	require.EqualValues(t, newDataLen, mb.Buffered())
	require.EqualValues(t, rbn, mb.ringBuffer.Buffered())

	bs, err := mb.Peek(-1)
	require.NoError(t, err)
	var p []byte
	for _, b := range bs {
		p = append(p, b...)
	}
	require.EqualValues(t, data, p)

	bs, err = mb.Peek(rbn)
	require.NoError(t, err)
	p = bs[0]
	require.EqualValues(t, data[:rbn], p)
	n, err = mb.Discard(rbn)
	require.NoError(t, err)
	require.EqualValues(t, rbn, n)
	require.NotNil(t, mb.ringBuffer)
	bs, err = mb.Peek(newDataLen - rbn)
	require.NoError(t, err)
	p = bs[0]
	require.EqualValues(t, data[rbn:], p)
	n, err = mb.Discard(newDataLen - rbn)
	require.NoError(t, err)
	require.EqualValues(t, newDataLen-rbn, n)
	require.True(t, mb.IsEmpty())

	runtime.GC() // release ring-buffer from pool.
	const maxBlocks = 100
	var (
		headCum int
		cum     int
		buf     bytes.Buffer
	)
	bs = bs[:0]
	for i := 0; i < maxBlocks; i++ {
		n := rand.Intn(512) + 128
		cum += n
		data := make([]byte, n)
		_, err := crand.Read(data)
		require.NoError(t, err)
		buf.Write(data)
		if i < 3 {
			headCum += n
			_, _ = mb.Write(data)
		} else {
			bs = append(bs, data)
		}
	}
	n, err = mb.Writev(bs)
	require.GreaterOrEqual(t, mb.ringBuffer.Len(), maxStaticSize)
	require.NoError(t, err)
	require.EqualValues(t, cum-headCum, n)
	require.EqualValues(t, cum, mb.Buffered())
	bs, err = mb.Peek(-1)
	require.NoError(t, err)
	p = p[:0]
	for _, b := range bs {
		p = append(p, b...)
	}
	require.EqualValues(t, buf.Bytes(), p)
	p = make([]byte, cum)
	n, err = mb.Read(p)
	require.NoError(t, err)
	require.EqualValues(t, cum, n)
	require.EqualValues(t, buf.Bytes(), p)

	require.NotNil(t, mb.ringBuffer)
	require.True(t, mb.IsEmpty())
}

func TestMixedBuffer_ReadFrom(t *testing.T) {
	const maxStaticSize = 2 * 1024
	mb, _ := New(maxStaticSize)
	const dataLen = 2 * 1024
	data := make([]byte, dataLen)
	_, err := crand.Read(data)
	require.NoError(t, err)
	r := bytes.NewReader(data)
	n, err := mb.ReadFrom(r)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, n)
	require.EqualValues(t, dataLen, mb.Buffered())
	newData := make([]byte, dataLen)
	_, err = crand.Read(newData)
	require.NoError(t, err)
	r.Reset(newData)
	n, err = mb.ReadFrom(r)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, n)
	require.EqualValues(t, 2*dataLen, mb.Buffered())
	require.False(t, mb.listBuffer.IsEmpty())

	buf := make([]byte, dataLen)
	var m int
	m, err = mb.Read(buf)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, m)
	require.EqualValues(t, data, buf)
	bs, err := mb.Peek(dataLen)
	require.NoError(t, err)
	var p []byte
	for _, b := range bs {
		p = append(p, b...)
	}
	require.EqualValues(t, newData, p)
	m, err = mb.Discard(dataLen)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, m)

	require.NotNil(t, mb.ringBuffer)
	require.True(t, mb.IsEmpty())
}

func TestMixedBuffer_WriteTo(t *testing.T) {
	const maxStaticSize = 4 * 1024
	mb, _ := New(maxStaticSize)
	const maxBlocks = 50
	var (
		headCum int
		cum     int
		bs      [][]byte
		buf     bytes.Buffer
	)

	for i := 0; i < maxBlocks; i++ {
		n := rand.Intn(512) + 128
		cum += n
		data := make([]byte, n)
		_, err := crand.Read(data)
		require.NoError(t, err)
		buf.Write(data)
		if i < 3 {
			headCum += n
			_, _ = mb.Write(data)
		} else {
			bs = append(bs, data)
		}
	}
	n, err := mb.Writev(bs)
	require.NoError(t, err)
	require.EqualValues(t, cum-headCum, n)
	require.EqualValues(t, cum, mb.Buffered())

	newBuf := bytes.NewBuffer(nil)
	var m int64
	m, err = mb.WriteTo(newBuf)
	require.NoError(t, err)
	require.EqualValues(t, cum, m)
	require.EqualValues(t, buf.Bytes(), newBuf.Bytes())

	require.NotNil(t, mb.ringBuffer)
	require.True(t, mb.IsEmpty())
}
