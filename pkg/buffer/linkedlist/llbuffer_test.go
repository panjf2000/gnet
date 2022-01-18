// Copyright (c) 2022 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linkedlist

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLinkedListBuffer_Basic(t *testing.T) {
	const maxBlocks = 100
	var (
		llb Buffer
		cum int
		buf bytes.Buffer
	)
	rand.Seed(time.Now().Unix())
	for i := 0; i < maxBlocks; i++ {
		n := rand.Intn(1024) + 128
		cum += n
		data := make([]byte, n)
		rand.Read(data)
		llb.PushBack(data)
		buf.Write(data)
	}
	require.EqualValues(t, maxBlocks, llb.Len())
	require.EqualValues(t, cum, llb.Buffered())

	bs := llb.Peek(cum / 4)
	var p []byte
	for _, b := range bs {
		p = append(p, b...)
	}
	pn := len(p)
	require.GreaterOrEqual(t, pn, cum/4)
	require.EqualValues(t, buf.Bytes()[:pn], p)
	tmpA := make([]byte, cum/16)
	tmpB := make([]byte, cum/16)
	rand.Read(tmpA)
	rand.Read(tmpB)
	bs = llb.PeekWithBytes(cum/4, tmpA, tmpB)
	p = p[:0]
	for _, b := range bs {
		p = append(p, b...)
	}
	pn = len(p)
	require.GreaterOrEqual(t, pn, cum/4)
	var tmpBuf bytes.Buffer
	tmpBuf.Write(tmpA)
	tmpBuf.Write(tmpB)
	tmpBuf.Write(buf.Bytes()[:pn-len(tmpA)-len(tmpB)])
	require.EqualValues(t, tmpBuf.Bytes(), p)

	pn, _ = llb.Discard(pn)
	buf.Next(pn)
	p = make([]byte, cum-pn)
	n, err := llb.Read(p)
	require.NoError(t, err)
	require.EqualValues(t, cum-pn, n)
	require.EqualValues(t, buf.Bytes(), p)
	require.True(t, llb.IsEmpty())
}

func TestLinkedListBuffer_ReadFrom(t *testing.T) {
	var llb Buffer
	const dataLen = 4 * 1024
	data := make([]byte, dataLen)
	rand.Seed(time.Now().Unix())
	rand.Read(data)
	r := bytes.NewReader(data)
	n, err := llb.ReadFrom(r)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, n)
	require.EqualValues(t, dataLen, llb.Buffered())

	llb.Reset()
	const headLen = 256
	head := make([]byte, headLen)
	rand.Read(head)
	llb.PushBack(head)
	rand.Read(data)
	r.Reset(data)
	n, err = llb.ReadFrom(r)
	require.NoError(t, err)
	require.EqualValues(t, dataLen, n)
	require.EqualValues(t, headLen+dataLen, llb.Buffered())
	buf := make([]byte, headLen+dataLen)
	var m int
	m, err = llb.Read(buf)
	require.NoError(t, err)
	require.EqualValues(t, headLen+dataLen, m)
	require.EqualValues(t, append(head, data...), buf)
	require.True(t, llb.IsEmpty())
}

func TestLinkedListBuffer_WriteTo(t *testing.T) {
	const maxBlocks = 20
	var (
		llb Buffer
		cum int
		buf bytes.Buffer
	)
	rand.Seed(time.Now().Unix())
	for i := 0; i < maxBlocks; i++ {
		n := rand.Intn(1024) + 128
		cum += n
		data := make([]byte, n)
		rand.Read(data)
		llb.PushBack(data)
		buf.Write(data)
	}
	require.EqualValues(t, maxBlocks, llb.Len())
	require.EqualValues(t, cum, llb.Buffered())

	newBuf := bytes.NewBuffer(nil)
	n, err := llb.WriteTo(newBuf)
	require.NoError(t, err)
	require.EqualValues(t, cum, n)
	require.EqualValues(t, buf.Bytes(), newBuf.Bytes())

	llb.Reset()
	buf.Reset()
	newBuf.Reset()
	cum = 0
	for i := 0; i < maxBlocks; i++ {
		n := rand.Intn(1024) + 128
		cum += n
		data := make([]byte, n)
		rand.Read(data)
		llb.PushBack(data)
		buf.Write(data)
	}
	require.EqualValues(t, maxBlocks, llb.Len())
	require.EqualValues(t, cum, llb.Buffered())

	var discarded int
	discarded, err = llb.Discard(cum / 2)
	require.NoError(t, err)
	buf.Next(discarded)
	n, err = llb.WriteTo(newBuf)
	require.NoError(t, err)
	require.EqualValues(t, cum-discarded, n)
	require.EqualValues(t, buf.Bytes(), newBuf.Bytes())
	llb.Reset()
	buf.Reset()
	newBuf.Reset()
}
