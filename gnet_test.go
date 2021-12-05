// Copyright (c) 2019 Andy Pan
// Copyright (c) 2017 Joshua J Baker
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

package gnet

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/panjf2000/gnet/pkg/errors"
	"github.com/panjf2000/gnet/pkg/logging"
	"github.com/panjf2000/gnet/pkg/pool/bytebuffer"
	"github.com/panjf2000/gnet/pkg/pool/goroutine"
)

var (
	packetLen = 1024
	streamLen = 1024 * 1024
)

func TestCodecServe(t *testing.T) {
	// start a server
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the server will be random sizes. 0KB - 1MB.
	// the server will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9991", false, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9992", false, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9993", false, false, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9994", false, false, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9995", true, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9996", true, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9997", true, false, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9998", true, false, 10, false, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9991", false, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9992", false, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9993", false, true, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9994", false, true, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9995", true, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9996", true, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9997", true, true, 10, false, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9998", true, true, 10, false, nil)
			})
		})
	})
	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9991", false, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9992", false, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9993", false, false, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9994", false, false, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9995", true, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9996", true, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9997", true, false, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9998", true, false, 10, true, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9991", false, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9992", false, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9993", false, true, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9994", false, true, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9995", true, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9996", true, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9997", true, true, 10, true, NewFixedLengthFrameCodec(packetLen))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe(t, "tcp", ":9998", true, true, 10, true, nil)
			})
		})
	})
}

type testCodecServer struct {
	*EventServer
	tester       *testing.T
	network      string
	addr         string
	multicore    bool
	async        bool
	nclients     int
	started      int32
	connected    int32
	disconnected int32
	codec        ICodec
	workerPool   *goroutine.Pool
}

func (s *testCodecServer) OnOpened(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	out = []byte("sweetness\r\n")
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testCodecServer) OnClosed(c Conn, err error) (action Action) {
	require.Equal(s.tester, c.Context(), c, "invalid context")

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
	}

	return
}

func (s *testCodecServer) React(packet []byte, c Conn) (out []byte, action Action) {
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

func (s *testCodecServer) Tick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			go startCodecClient(s.tester, s.network, s.addr, s.multicore, s.async, s.codec)
		}
	}
	delay = time.Second / 5
	return
}

var (
	n            = 0
	fieldLengths = []int{1, 2, 3, 4, 8}
)

func testCodecServe(
	t *testing.T,
	network, addr string,
	multicore, async bool,
	nclients int,
	reuseport bool,
	codec ICodec,
) {
	var err error
	fieldLength := fieldLengths[n]
	if codec == nil {
		encoderConfig := EncoderConfig{
			ByteOrder:                       binary.BigEndian,
			LengthFieldLength:               fieldLength,
			LengthAdjustment:                0,
			LengthIncludesLengthFieldLength: false,
		}
		decoderConfig := DecoderConfig{
			ByteOrder:           binary.BigEndian,
			LengthFieldOffset:   0,
			LengthFieldLength:   fieldLength,
			LengthAdjustment:    0,
			InitialBytesToStrip: fieldLength,
		}
		codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	}
	n++
	if n > 4 {
		n = 0
	}
	ts := &testCodecServer{
		tester: t, network: network, addr: addr, multicore: multicore, async: async, nclients: nclients,
		codec: codec, workerPool: goroutine.Default(),
	}
	err = Serve(
		ts,
		network+"://"+addr,
		WithMulticore(multicore),
		WithTicker(true),
		WithReadBufferCap(8*1024),
		WithLogLevel(logging.DebugLevel),
		WithTCPKeepAlive(time.Minute*5),
		WithSocketRecvBuffer(8*1024),
		WithSocketSendBuffer(8*1024),
		WithCodec(codec),
		WithReusePort(reuseport),
	)
	assert.NoError(t, err)
}

func startCodecClient(t *testing.T, network, addr string, multicore, async bool, codec ICodec) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	require.NoError(t, err)
	require.Equal(t, string(msg), "sweetness\r\n")
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		// data := []byte("Hello, World")
		// data := make([]byte, 1024)
		// rand.Read(data)
		data := []byte(strings.Repeat("x", 1024))
		reqData, _ := codec.Encode(nil, data)
		_, err = c.Write(reqData)
		require.NoError(t, err)
		respData := make([]byte, len(reqData))
		_, err = io.ReadFull(rd, respData)
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

func TestServe(t *testing.T) {
	// start a server
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the server will be random sizes. 0KB - 1MB.
	// the server will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", false, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", false, false, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", false, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", false, false, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", false, false, false, true, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", false, false, true, true, true, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", false, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", false, false, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", false, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", false, false, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", false, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", false, false, true, false, false, 10, SourceAddrHash)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", false, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", false, false, true, true, false, 10, SourceAddrHash)
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", false, false, false, true, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", false, false, true, true, true, 10, SourceAddrHash)
			})
		})
	})

	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", true, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", true, false, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", true, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", true, false, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", true, false, false, true, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", true, false, true, true, true, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", true, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", true, false, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", true, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", true, false, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", true, false, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", true, false, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", true, false, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", true, false, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", true, false, false, true, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", true, false, true, true, true, 10, LeastConnections)
			})
		})
	})

	t.Run("poll-reuseaddr", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", false, true, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", false, true, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9991", false, true, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "tcp", ":9992", false, true, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", false, true, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", false, true, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "udp", ":9991", false, true, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "udp", ":9992", false, true, true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", false, true, false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", false, true, true, false, false, 10, LeastConnections)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet1.sock", false, true, false, true, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe(t, "unix", "gnet2.sock", false, true, true, true, false, 10, LeastConnections)
			})
		})
	})
}

type testServer struct {
	*EventServer
	tester       *testing.T
	svr          Server
	network      string
	addr         string
	multicore    bool
	async        bool
	writev       bool
	nclients     int
	started      int32
	connected    int32
	clientActive int32
	disconnected int32
	workerPool   *goroutine.Pool
}

func (s *testServer) OnInitComplete(svr Server) (action Action) {
	s.svr = svr
	return
}

func (s *testServer) OnOpened(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	out = []byte("sweetness\r\n")
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testServer) OnClosed(c Conn, err error) (action Action) {
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

func (s *testServer) React(packet []byte, c Conn) (out []byte, action Action) {
	if s.async {
		buf := bytebuffer.Get()
		_, _ = buf.Write(packet)

		if s.network == "tcp" || s.network == "unix" {
			// just for test
			_ = c.BufferLength()
			c.ShiftN(1)

			_ = s.workerPool.Submit(
				func() {
					if s.writev {
						mid := buf.Len() / 2
						bs := make([][]byte, 2)
						bs[0] = buf.B[:mid]
						bs[1] = buf.B[mid:]
						_ = c.AsyncWritev(bs)
					} else {
						_ = c.AsyncWrite(buf.Bytes())
					}
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

func (s *testServer) Tick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			go func() {
				startClient(s.tester, s.network, s.addr, s.multicore, s.async)
				atomic.AddInt32(&s.clientActive, -1)
			}()
		}
	}
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
	delay = time.Second / 5
	return
}

func testServe(t *testing.T, network, addr string, reuseport, reuseaddr, multicore, async, writev bool, nclients int, lb LoadBalancing) {
	ts := &testServer{
		tester:     t,
		network:    network,
		addr:       addr,
		multicore:  multicore,
		async:      async,
		writev:     writev,
		nclients:   nclients,
		workerPool: goroutine.Default(),
	}
	err := Serve(ts,
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

func startClient(t *testing.T, network, addr string, multicore, async bool) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	rd := bufio.NewReader(c)
	if network != "udp" {
		msg, err := rd.ReadBytes('\n')
		require.NoError(t, err)
		require.Equal(t, string(msg), "sweetness\r\n", "bad header")
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
		_, err = c.Write(reqData)
		require.NoError(t, err)
		respData := make([]byte, len(reqData))
		_, err = io.ReadFull(rd, respData)
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

func TestDefaultGnetServer(t *testing.T) {
	svr := EventServer{}
	svr.OnInitComplete(Server{})
	svr.OnOpened(nil)
	svr.OnClosed(nil, nil)
	svr.PreWrite(nil)
	svr.AfterWrite(nil, nil)
	svr.React(nil, nil)
	svr.Tick()
}

type testBadAddrServer struct {
	*EventServer
}

func (t *testBadAddrServer) OnInitComplete(srv Server) (action Action) {
	return Shutdown
}

func TestBadAddresses(t *testing.T) {
	events := new(testBadAddrServer)
	err := Serve(events, "tulip://howdy")
	assert.Error(t, err)
	err = Serve(events, "howdy")
	assert.Error(t, err)
	err = Serve(events, "tcp://")
	assert.NoError(t, err)
}

func TestTick(t *testing.T) {
	testTick("tcp", ":9989", t)
}

type testTickServer struct {
	*EventServer
	count int
}

func (t *testTickServer) Tick() (delay time.Duration, action Action) {
	if t.count == 25 {
		action = Shutdown
		return
	}
	t.count++
	delay = time.Millisecond * 10
	return
}

func testTick(network, addr string, t *testing.T) {
	events := &testTickServer{}
	start := time.Now()
	opts := Options{Ticker: true}
	err := Serve(events, network+"://"+addr, WithOptions(opts))
	assert.NoError(t, err)
	dur := time.Since(start)
	if dur < 250&time.Millisecond || dur > time.Second {
		t.Logf("bad ticker timing: %d", dur)
	}
}

func TestWakeConn(t *testing.T) {
	testWakeConn(t, "tcp", ":9990")
}

type testWakeConnServer struct {
	*EventServer
	tester  *testing.T
	network string
	addr    string
	conn    chan Conn
	c       Conn
	wake    bool
}

func (t *testWakeConnServer) OnOpened(c Conn) (out []byte, action Action) {
	t.conn <- c
	return
}

func (t *testWakeConnServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testWakeConnServer) React(packet []byte, c Conn) (out []byte, action Action) {
	out = []byte("Waking up.")
	action = -1
	return
}

func (t *testWakeConnServer) Tick() (delay time.Duration, action Action) {
	if !t.wake {
		t.wake = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			r := make([]byte, 10)
			_, err = conn.Read(r)
			require.NoError(t.tester, err)
		}()
		return
	}
	t.c = <-t.conn
	_ = t.c.Wake()
	delay = time.Millisecond * 100
	return
}

func testWakeConn(t *testing.T, network, addr string) {
	svr := &testWakeConnServer{tester: t, network: network, addr: addr, conn: make(chan Conn, 1)}
	logger := zap.NewExample()
	err := Serve(svr, network+"://"+addr, WithTicker(true), WithNumEventLoop(2*runtime.NumCPU()),
		WithLogger(logger.Sugar()))
	assert.NoError(t, err)
	_ = logger.Sync()
}

func TestShutdown(t *testing.T) {
	testShutdown(t, "tcp", ":9991")
}

type testShutdownServer struct {
	*EventServer
	tester  *testing.T
	network string
	addr    string
	count   int
	clients int64
	N       int
}

func (t *testShutdownServer) OnOpened(c Conn) (out []byte, action Action) {
	atomic.AddInt64(&t.clients, 1)
	return
}

func (t *testShutdownServer) OnClosed(c Conn, err error) (action Action) {
	atomic.AddInt64(&t.clients, -1)
	return
}

func (t *testShutdownServer) Tick() (delay time.Duration, action Action) {
	if t.count == 0 {
		// start clients
		for i := 0; i < t.N; i++ {
			go func() {
				conn, err := net.Dial(t.network, t.addr)
				require.NoError(t.tester, err)
				defer conn.Close()
				_, err = conn.Read([]byte{0})
				require.Error(t.tester, err)
			}()
		}
	} else if int(atomic.LoadInt64(&t.clients)) == t.N {
		action = Shutdown
	}
	t.count++
	delay = time.Second / 20
	return
}

func testShutdown(t *testing.T, network, addr string) {
	events := &testShutdownServer{tester: t, network: network, addr: addr, N: 10}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
	require.Equal(t, int(events.clients), 0, "did not call close on all clients")
}

func TestCloseActionError(t *testing.T) {
	testCloseActionError(t, "tcp", ":9992")
}

type testCloseActionErrorServer struct {
	*EventServer
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseActionErrorServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseActionErrorServer) React(packet []byte, c Conn) (out []byte, action Action) {
	out = packet
	action = Close
	return
}

func (t *testCloseActionErrorServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionError(t *testing.T, network, addr string) {
	events := &testCloseActionErrorServer{tester: t, network: network, addr: addr}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestShutdownActionError(t *testing.T) {
	testShutdownActionError(t, "tcp", ":9993")
}

type testShutdownActionErrorServer struct {
	*EventServer
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testShutdownActionErrorServer) React(packet []byte, c Conn) (out []byte, action Action) {
	c.ReadN(-1) // just for test
	out = packet
	action = Shutdown
	return
}

func (t *testShutdownActionErrorServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionError(t *testing.T, network, addr string) {
	events := &testShutdownActionErrorServer{tester: t, network: network, addr: addr}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestCloseActionOnOpen(t *testing.T) {
	testCloseActionOnOpen(t, "tcp", ":9994")
}

type testCloseActionOnOpenServer struct {
	*EventServer
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseActionOnOpenServer) OnOpened(c Conn) (out []byte, action Action) {
	action = Close
	return
}

func (t *testCloseActionOnOpenServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseActionOnOpenServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionOnOpen(t *testing.T, network, addr string) {
	events := &testCloseActionOnOpenServer{tester: t, network: network, addr: addr}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestShutdownActionOnOpen(t *testing.T) {
	testShutdownActionOnOpen(t, "tcp", ":9995")
}

type testShutdownActionOnOpenServer struct {
	*EventServer
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testShutdownActionOnOpenServer) OnOpened(c Conn) (out []byte, action Action) {
	action = Shutdown
	return
}

func (t *testShutdownActionOnOpenServer) OnShutdown(s Server) {
	dupFD, err := s.DupFd()
	logging.Debugf("dup fd: %d with error: %v\n", dupFD, err)
}

func (t *testShutdownActionOnOpenServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionOnOpen(t *testing.T, network, addr string) {
	events := &testShutdownActionOnOpenServer{tester: t, network: network, addr: addr}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestUDPShutdown(t *testing.T) {
	testUDPShutdown(t, "udp4", ":9000")
}

type testUDPShutdownServer struct {
	*EventServer
	tester  *testing.T
	network string
	addr    string
	tick    bool
}

func (t *testUDPShutdownServer) React(packet []byte, c Conn) (out []byte, action Action) {
	out = packet
	action = Shutdown
	return
}

func (t *testUDPShutdownServer) Tick() (delay time.Duration, action Action) {
	if !t.tick {
		t.tick = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, err = conn.Write(data)
			require.NoError(t.tester, err)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testUDPShutdown(t *testing.T, network, addr string) {
	svr := &testUDPShutdownServer{tester: t, network: network, addr: addr}
	err := Serve(svr, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestCloseConnection(t *testing.T) {
	testCloseConnection(t, "tcp", ":9996")
}

type testCloseConnectionServer struct {
	*EventServer
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseConnectionServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseConnectionServer) React(packet []byte, c Conn) (out []byte, action Action) {
	out = packet
	go func() {
		time.Sleep(time.Second)
		_ = c.Close()
	}()
	return
}

func (t *testCloseConnectionServer) Tick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
			// waiting the server shutdown.
			_, err = conn.Read(data)
			require.Error(t.tester, err)
		}()
		return
	}
	return
}

func testCloseConnection(t *testing.T, network, addr string) {
	events := &testCloseConnectionServer{tester: t, network: network, addr: addr}
	err := Serve(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestServerOptionsCheck(t *testing.T) {
	err := Serve(&EventServer{}, "tcp://:3500", WithNumEventLoop(10001), WithLockOSThread(true))
	assert.EqualError(t, err, errors.ErrTooManyEventLoopThreads.Error(), "error returned with LockOSThread option")
}

func TestStop(t *testing.T) {
	testStop(t, "tcp", ":9997")
}

type testStopServer struct {
	*EventServer
	tester                   *testing.T
	network, addr, protoAddr string
	action                   bool
}

func (t *testStopServer) OnClosed(c Conn, err error) (action Action) {
	logging.Debugf("closing connection...")
	return
}

func (t *testStopServer) React(packet []byte, c Conn) (out []byte, action Action) {
	out = packet
	return
}

func (t *testStopServer) Tick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				logging.Debugf("stop server...", Stop(ctx, t.protoAddr))
			}()

			// waiting the server shutdown.
			_, err = conn.Read(data)
			require.Error(t.tester, err)
		}()
		return
	}
	return
}

func testStop(t *testing.T, network, addr string) {
	events := &testStopServer{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr}
	err := Serve(events, events.protoAddr, WithTicker(true))
	assert.NoError(t, err)
}

// Test should not panic when we wake-up server_closed conn.
func TestClosedWakeUp(t *testing.T) {
	events := &testClosedWakeUpServer{
		tester:      t,
		EventServer: &EventServer{}, network: "tcp", addr: ":8888", protoAddr: "tcp://:8888",
		clientClosed: make(chan struct{}),
		serverClosed: make(chan struct{}),
		wakeup:       make(chan struct{}),
	}

	err := Serve(events, events.protoAddr)
	assert.NoError(t, err)
}

type testClosedWakeUpServer struct {
	*EventServer
	tester                   *testing.T
	network, addr, protoAddr string

	wakeup       chan struct{}
	serverClosed chan struct{}
	clientClosed chan struct{}
}

func (tes *testClosedWakeUpServer) OnInitComplete(_ Server) (action Action) {
	go func() {
		c, err := net.Dial(tes.network, tes.addr)
		require.NoError(tes.tester, err)

		_, err = c.Write([]byte("hello"))
		require.NoError(tes.tester, err)

		<-tes.wakeup
		_, err = c.Write([]byte("hello again"))
		require.NoError(tes.tester, err)

		close(tes.clientClosed)
		<-tes.serverClosed

		logging.Debugf("stop server...", Stop(context.TODO(), tes.protoAddr))
	}()

	return None
}

func (tes *testClosedWakeUpServer) React(_ []byte, conn Conn) ([]byte, Action) {
	require.NotNil(tes.tester, conn.RemoteAddr())

	select {
	case <-tes.wakeup:
	default:
		close(tes.wakeup)
	}

	// Actually goroutines here needed only on windows since its async actions
	// rely on an unbuffered channel and since we already into it - this will
	// block forever.
	go func() { require.NoError(tes.tester, conn.Wake()) }()
	go func() { require.NoError(tes.tester, conn.Close()) }()

	<-tes.clientClosed

	return []byte("answer"), None
}

func (tes *testClosedWakeUpServer) OnClosed(c Conn, err error) (action Action) {
	select {
	case <-tes.serverClosed:
	default:
		close(tes.serverClosed)
	}
	return
}
