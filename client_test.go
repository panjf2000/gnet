// Copyright (c) 2021 Andy Pan
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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"encoding/binary"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/panjf2000/gnet/pkg/logging"
	bbPool "github.com/panjf2000/gnet/pkg/pool/bytebuffer"
	goPool "github.com/panjf2000/gnet/pkg/pool/goroutine"
)

type clientEvents struct {
	*EventServer
	svr       *testClientServer
	packetLen int
	rspChMap  sync.Map
}

func (ev *clientEvents) OnOpened(c Conn) ([]byte, Action) {
	c.SetContext([]byte{})
	rspCh := make(chan []byte, 1)
	ev.rspChMap.Store(c.LocalAddr().String(), rspCh)
	return nil, None
}

func (ev *clientEvents) OnClosed(c Conn, err error) Action {
	if ev.svr != nil {
		if atomic.AddInt32(&ev.svr.clientActive, -1) == 0 {
			return Shutdown
		}
	}
	return None
}

func (ev *clientEvents) React(packet []byte, c Conn) (out []byte, action Action) {
	ctx := c.Context()
	var p []byte
	if ctx != nil {
		p = ctx.([]byte)
	} else { // UDP
		ev.packetLen = 1024
	}
	p = append(p, packet...)
	if len(p) < ev.packetLen {
		c.SetContext(p)
		return
	}
	v, _ := ev.rspChMap.Load(c.LocalAddr().String())
	rspCh := v.(chan []byte)
	rspCh <- p
	c.SetContext([]byte{})
	return
}

func (ev *clientEvents) Tick() (delay time.Duration, action Action) {
	delay = 200 * time.Millisecond
	return
}

func TestCodecServeWithGnetClient(t *testing.T) {
	// start a server
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the server will be random sizes. 0KB - 1MB.
	// the server will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9991", false, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9992", false, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, false, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9994", false, false, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9995", true, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9996", true, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, false, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9998", true, false, 10, false, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9991", false, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9992", false, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, true, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9994", false, true, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9995", true, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9996", true, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, true, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9998", true, true, 10, false, nil)
			})
		})
	})
	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9991", false, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9992", false, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, false, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9994", false, false, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9995", true, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9996", true, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, false, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9998", true, false, 10, true, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9991", false, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9992", false, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, true, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9994", false, true, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9995", true, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9996", true, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, true, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServeWithGnetClient(t, "tcp", ":9998", true, true, 10, true, nil)
			})
		})
	})
}

type testCodecClientServer struct {
	*EventServer
	client       *Client
	clientEV     *clientEvents
	nclients     int
	tester       *testing.T
	network      string
	addr         string
	multicore    bool
	async        bool
	started      int32
	connected    int32
	disconnected int32
	codec        ICodec
	workerPool   *goPool.Pool
}

func (s *testCodecClientServer) OnOpened(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testCodecClientServer) OnClosed(c Conn, err error) (action Action) {
	require.Equal(s.tester, c.Context(), c, "invalid context")
	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
	}

	return
}

func (s *testCodecClientServer) React(packet []byte, c Conn) (out []byte, action Action) {
	if s.async {
		if packet != nil {
			data := append([]byte{}, packet...)
			_ = s.workerPool.Submit(func() {
				_ = c.AsyncWrite(data)
			})
		}
		return
	}
	out = packet
	return
}

func (s *testCodecClientServer) Tick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			go startCodecGnetClient(s.tester, s.client, s.clientEV, s.network, s.addr, s.multicore, s.async, s.codec)
		}
	}
	delay = time.Second / 5
	return
}

func testCodecServeWithGnetClient(
	t *testing.T,
	network, addr string,
	multicore, async bool,
	nclients int,
	reuseport bool,
	codec ICodec,
) {
	var err error
	if codec == nil {
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
		codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	}
	ts := &testCodecClientServer{
		tester: t, network: network, addr: addr, multicore: multicore, async: async,
		codec: codec, workerPool: goPool.Default(), nclients: nclients,
	}
	ts.clientEV = &clientEvents{packetLen: packetLen}
	ts.client, err = NewClient(
		ts.clientEV,
		WithLogLevel(logging.DebugLevel),
		WithCodec(codec),
		WithReadBufferCap(8*1024),
		WithLockOSThread(true),
		WithTCPNoDelay(TCPNoDelay),
		WithTCPKeepAlive(time.Minute),
		WithSocketRecvBuffer(8*1024),
		WithSocketSendBuffer(8*1024),
		WithTicker(true),
	)
	assert.NoError(t, err)
	err = ts.client.Start()
	assert.NoError(t, err)
	err = Serve(
		ts,
		network+"://"+addr,
		WithMulticore(multicore),
		WithTicker(true),
		WithLogLevel(logging.DebugLevel),
		WithSocketRecvBuffer(8*1024),
		WithSocketSendBuffer(8*1024),
		WithCodec(codec),
		WithReusePort(reuseport),
	)
	assert.NoError(t, err)
	err = ts.client.Stop()
	assert.NoError(t, err)
}

func startCodecGnetClient(t *testing.T, cli *Client, ev *clientEvents, network, addr string, multicore, async bool, codec ICodec) {
	c, err := cli.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	var (
		ok bool
		v  interface{}
	)
	start := time.Now()
	for time.Since(start) < time.Second {
		v, ok = ev.rspChMap.Load(c.LocalAddr().String())
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, ok)
	rspCh := v.(chan []byte)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start = time.Now()
	for time.Since(start) < duration {
		// reqData := make([]byte, 1024)
		// rand.Read(reqData)
		reqData := []byte(strings.Repeat("x", packetLen))
		err = c.AsyncWrite(reqData)
		require.NoError(t, err)
		respData := <-rspCh
		if !async {
			// require.Equalf(t, reqData, respData, "response mismatch with protocol:%s, multi-core:%t, content of bytes: %d vs %d", network, multicore, string(reqData), string(respData))
			require.Equalf(
				t,
				reqData,
				respData,
				"response mismatch with protocol:%s, multi-core:%t, length of bytes: %d vs %d",
				network,
				multicore,
				len(reqData),
				len(respData),
			)
		}
	}
}

func TestServeWithGnetClient(t *testing.T) {
	// start a server
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the server will be random sizes. 0KB - 1MB.
	// the server will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9991", false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9992", false, false, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9991", false, false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9992", false, false, true, true, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9991", false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9992", false, false, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9991", false, false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9992", false, false, true, true, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet1.sock", false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet2.sock", false, false, true, false, 10, SourceAddrHash)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet1.sock", false, false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet2.sock", false, false, true, true, 10, SourceAddrHash)
			})
		})
	})

	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9991", true, true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9992", true, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9991", true, true, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "tcp", ":9992", true, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9991", true, true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9992", true, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9991", true, true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "udp", ":9992", true, true, true, true, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet1.sock", true, true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet2.sock", true, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet1.sock", true, true, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServeWithGnetClient(t, "unix", "gnet2.sock", true, true, true, true, 10, LeastConnections)
			})
		})
	})
}

type testClientServer struct {
	*EventServer
	client       *Client
	clientEV     *clientEvents
	tester       *testing.T
	svr          Server
	network      string
	addr         string
	multicore    bool
	async        bool
	nclients     int
	started      int32
	connected    int32
	clientActive int32
	disconnected int32
	workerPool   *goPool.Pool
}

func (s *testClientServer) OnInitComplete(svr Server) (action Action) {
	s.svr = svr
	return
}

func (s *testClientServer) OnOpened(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testClientServer) OnClosed(c Conn, err error) (action Action) {
	if err != nil {
		logging.Debugf("error occurred on closed, %v\n", err)
	}
	if s.network != "udp" {
		require.Equal(s.tester, c.Context(), c, "invalid context")
	}

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
		s.workerPool.Release()
	}

	return
}

func (s *testClientServer) React(packet []byte, c Conn) (out []byte, action Action) {
	if s.async {
		buf := bbPool.Get()
		_, _ = buf.Write(packet)

		if s.network == "tcp" || s.network == "unix" {
			// just for test
			_ = c.BufferLength()
			c.ShiftN(1)

			_ = s.workerPool.Submit(
				func() {
					_ = c.AsyncWrite(buf.Bytes())
				})
			return
		} else if s.network == "udp" {
			_ = s.workerPool.Submit(
				func() {
					_ = c.SendTo(buf.Bytes())
				})
			return
		}
		return
	}
	out = packet
	return
}

func (s *testClientServer) Tick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			go startGnetClient(s.tester, s.client, s.clientEV, s.network, s.addr, s.multicore, s.async)
		}
	}
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
	delay = time.Second / 5
	return
}

func testServeWithGnetClient(t *testing.T, network, addr string, reuseport, reuseaddr, multicore, async bool, nclients int, lb LoadBalancing) {
	ts := &testClientServer{
		tester:     t,
		network:    network,
		addr:       addr,
		multicore:  multicore,
		async:      async,
		nclients:   nclients,
		workerPool: goPool.Default(),
	}
	var err error
	ts.clientEV = &clientEvents{packetLen: streamLen, svr: ts}
	ts.client, err = NewClient(
		ts.clientEV,
		WithLogLevel(logging.DebugLevel),
		WithLockOSThread(true),
		WithTicker(true),
	)
	assert.NoError(t, err)
	err = ts.client.Start()
	assert.NoError(t, err)
	err = Serve(ts,
		network+"://"+addr,
		WithLockOSThread(async),
		WithMulticore(multicore),
		WithReusePort(reuseport),
		WithReuseAddr(reuseaddr),
		WithTicker(true),
		WithTCPKeepAlive(time.Minute*1),
		WithTCPNoDelay(TCPDelay),
		WithLoadBalancing(lb))
	assert.NoError(t, err)
}

func startGnetClient(t *testing.T, cli *Client, ev *clientEvents, network, addr string, multicore, async bool) {
	rand.Seed(time.Now().UnixNano())
	c, err := cli.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	var rspCh chan []byte
	if network == "udp" {
		rspCh = make(chan []byte, 1)
		ev.rspChMap.Store(c.LocalAddr().String(), rspCh)
	} else {
		var (
			v  interface{}
			ok bool
		)
		start := time.Now()
		for time.Since(start) < time.Second {
			v, ok = ev.rspChMap.Load(c.LocalAddr().String())
			if ok {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		require.True(t, ok)
		rspCh = v.(chan []byte)
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, streamLen)
		if network == "udp" {
			reqData = reqData[:1024]
		}
		_, err = rand.Read(reqData)
		require.NoError(t, err)
		if network == "udp" {
			err = c.SendTo(reqData)
		} else {
			err = c.AsyncWrite(reqData)
		}
		require.NoError(t, err)
		respData := <-rspCh
		require.NoError(t, err)
		if !async {
			// require.Equalf(t, reqData, respData, "response mismatch with protocol:%s, multi-core:%t, content of bytes: %d vs %d", network, multicore, string(reqData), string(respData))
			require.Equalf(
				t,
				reqData,
				respData,
				"response mismatch with protocol:%s, multi-core:%t, length of bytes: %d vs %d",
				network,
				multicore,
				len(reqData),
				len(respData),
			)
		}
	}
}
