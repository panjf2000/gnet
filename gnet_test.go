// Copyright (c) 2019 Andy Pan
// Copyright (c) 2017 Joshua J Baker
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
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/panjf2000/gnet/errors"
	"github.com/panjf2000/gnet/pool/bytebuffer"
	"github.com/panjf2000/gnet/pool/goroutine"
	"go.uber.org/zap"
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
				testCodecServe("tcp", ":9991", false, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9992", false, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9993", false, false, 10, false, NewFixedLengthFrameCodec(64))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9994", false, false, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9995", true, false, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9996", true, false, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9997", true, false, 10, false, NewFixedLengthFrameCodec(64))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9998", true, false, 10, false, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9991", false, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9992", false, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9993", false, true, 10, false, NewFixedLengthFrameCodec(64))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9994", false, true, 10, false, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9995", true, true, 10, false, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9996", true, true, 10, false, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9997", true, true, 10, false, NewFixedLengthFrameCodec(64))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9998", true, true, 10, false, nil)
			})
		})
	})
	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9991", false, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9992", false, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9993", false, false, 10, true, NewFixedLengthFrameCodec(64))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9994", false, false, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9995", true, false, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9996", true, false, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9997", true, false, 10, true, NewFixedLengthFrameCodec(64))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9998", true, false, 10, true, nil)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9991", false, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("1-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9992", false, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("1-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9993", false, true, 10, true, NewFixedLengthFrameCodec(64))
			})
			t.Run("1-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9994", false, true, 10, true, nil)
			})
			t.Run("N-loop-LineBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9995", true, true, 10, true, new(LineBasedFrameCodec))
			})
			t.Run("N-loop-DelimiterBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9996", true, true, 10, true, NewDelimiterBasedFrameCodec('|'))
			})
			t.Run("N-loop-FixedLengthFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9997", true, true, 10, true, NewFixedLengthFrameCodec(64))
			})
			t.Run("N-loop-LengthFieldBasedFrameCodec", func(t *testing.T) {
				testCodecServe("tcp", ":9998", true, true, 10, true, nil)
			})
		})
	})
}

type testCodecServer struct {
	*EventServer
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
	if c.LocalAddr() == nil {
		panic("nil local addr")
	}
	if c.RemoteAddr() == nil {
		panic("nil local addr")
	}
	return
}

func (s *testCodecServer) OnClosed(c Conn, err error) (action Action) {
	if c.Context() != c {
		panic("invalid context")
	}

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
	}

	return
}

func (s *testCodecServer) React(frame []byte, c Conn) (out []byte, action Action) {
	if s.async {
		if frame != nil {
			data := append([]byte{}, frame...)
			_ = s.workerPool.Submit(func() {
				_ = c.AsyncWrite(data)
			})
		}
		return
	}
	out = frame
	return
}

func (s *testCodecServer) Tick() (delay time.Duration, action Action) {
	if atomic.LoadInt32(&s.started) == 0 {
		for i := 0; i < s.nclients; i++ {
			go func() {
				startCodecClient(s.network, s.addr, s.multicore, s.async, s.codec)
			}()
		}
		atomic.StoreInt32(&s.started, 1)
	}
	delay = time.Second / 5
	return
}

var (
	n            = 0
	fieldLengths = []int{1, 2, 3, 4, 8}
)

func testCodecServe(network, addr string, multicore, async bool, nclients int, reuseport bool, codec ICodec) {
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
		network: network, addr: addr, multicore: multicore, async: async, nclients: nclients,
		codec: codec, workerPool: goroutine.Default(),
	}
	err = Serve(ts, network+"://"+addr, WithMulticore(multicore), WithTicker(true),
		WithTCPKeepAlive(time.Minute*5), WithSocketRecvBuffer(8*1024), WithSocketSendBuffer(8*1024), WithCodec(codec), WithReusePort(reuseport))
	if err != nil {
		panic(err)
	}
}

func startCodecClient(network, addr string, multicore, async bool, codec ICodec) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	if err != nil {
		panic(err)
	}
	if string(msg) != "sweetness\r\n" {
		panic("bad header")
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		// data := []byte("Hello, World")
		data := make([]byte, 1024)
		rand.Read(data)
		encodedData, _ := codec.Encode(nil, data)
		if _, err := c.Write(encodedData); err != nil {
			panic(err)
		}
		data2 := make([]byte, len(encodedData))
		if _, err := io.ReadFull(rd, data2); err != nil {
			panic(err)
		}
		if !bytes.Equal(encodedData, data2) && !async {
			// panic(fmt.Sprintf("mismatch %s/multi-core:%t: %d vs %d bytes, %s:%s", network, multicore,
			//	len(encodedData), len(data2), string(encodedData), string(data2)))
			panic(fmt.Sprintf("mismatch %s/multi-core:%t: %d vs %d bytes", network, multicore, len(encodedData), len(data2)))
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
				testServe("tcp", ":9991", false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, true, true, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9991", false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9992", false, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9991", false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9992", false, true, true, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("unix", "gnet1.sock", false, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("unix", "gnet2.sock", false, true, false, 10, SourceAddrHash)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("unix", "gnet1.sock", false, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("unix", "gnet2.sock", false, true, true, 10, SourceAddrHash)
			})
		})
	})

	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", true, true, false, 10, LeastConnections)
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", true, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", true, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9991", true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9992", true, true, false, 10, LeastConnections)
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9991", true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9992", true, true, true, 10, LeastConnections)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("unix", "gnet1.sock", true, false, false, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("unix", "gnet2.sock", true, true, false, 10, LeastConnections)
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("unix", "gnet1.sock", true, false, true, 10, RoundRobin)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("unix", "gnet2.sock", true, true, true, 10, LeastConnections)
			})
		})
	})
}

type testServer struct {
	*EventServer
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
	if c.LocalAddr() == nil {
		panic("nil local addr")
	}
	if c.RemoteAddr() == nil {
		panic("nil local addr")
	}
	return
}

func (s *testServer) OnClosed(c Conn, err error) (action Action) {
	if err != nil {
		fmt.Printf("error occurred on closed, %v\n", err)
	}
	if c.Context() != c {
		panic("invalid context")
	}

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
		s.workerPool.Release()
	}

	return
}

func (s *testServer) React(frame []byte, c Conn) (out []byte, action Action) {
	if s.async {
		buf := bytebuffer.Get()
		_, _ = buf.Write(frame)

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
	out = frame
	return
}

func (s *testServer) Tick() (delay time.Duration, action Action) {
	if atomic.LoadInt32(&s.started) == 0 {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			go func() {
				startClient(s.network, s.addr, s.multicore, s.async)
				atomic.AddInt32(&s.clientActive, -1)
			}()
		}
		atomic.StoreInt32(&s.started, 1)
	}
	fmt.Printf("active connections: %d\n", s.svr.CountConnections())
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
	delay = time.Second / 5
	return
}

func testServe(network, addr string, reuseport, multicore, async bool, nclients int, lb LoadBalancing) {
	ts := &testServer{
		network:    network,
		addr:       addr,
		multicore:  multicore,
		async:      async,
		nclients:   nclients,
		workerPool: goroutine.Default(),
	}
	must(Serve(ts, network+"://"+addr, WithLockOSThread(async), WithMulticore(multicore), WithReusePort(reuseport), WithTicker(true),
		WithTCPKeepAlive(time.Minute*1), WithTCPNoDelay(TCPDelay), WithLoadBalancing(lb)))
}

func startClient(network, addr string, multicore, async bool) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	rd := bufio.NewReader(c)
	if network != "udp" {
		msg, err := rd.ReadBytes('\n')
		if err != nil {
			panic(err)
		}
		if string(msg) != "sweetness\r\n" {
			panic("bad header")
		}
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		// sz := rand.Intn(10) * (1024 * 1024)
		sz := 1024 * 1024
		data := make([]byte, sz)
		if network == "udp" || network == "unix" {
			n := 1024
			data = data[:n]
		}
		if _, err := rand.Read(data); err != nil {
			panic(err)
		}
		if _, err := c.Write(data); err != nil {
			panic(err)
		}
		data2 := make([]byte, len(data))
		if _, err := io.ReadFull(rd, data2); err != nil {
			panic(err)
		}
		if !bytes.Equal(data, data2) && !async {
			panic(fmt.Sprintf("mismatch %s/multi-core:%t: %d vs %d bytes\n", network, multicore, len(data), len(data2)))
		}
	}
}

func must(err error) {
	if err != nil && err != errors.ErrUnsupportedProtocol {
		panic(err)
	}
}

func TestDefaultGnetServer(t *testing.T) {
	svr := EventServer{}
	svr.OnInitComplete(Server{})
	svr.OnOpened(nil)
	svr.OnClosed(nil, nil)
	svr.PreWrite()
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
	if err := Serve(events, "tulip://howdy"); err == nil {
		t.Fatalf("expected error")
	}
	if err := Serve(events, "howdy"); err == nil {
		t.Fatalf("expected error")
	}
	if err := Serve(events, "tcp://"); err != nil {
		t.Fatalf("expected nil, got '%v'", err)
	}
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
	must(Serve(events, network+"://"+addr, WithOptions(opts)))
	dur := time.Since(start)
	if dur < 250&time.Millisecond || dur > time.Second {
		t.Logf("bad ticker timing: %d", dur)
	}
}

func TestWakeConn(t *testing.T) {
	testWakeConn("tcp", ":9990")
}

type testWakeConnServer struct {
	*EventServer
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

func (t *testWakeConnServer) React(frame []byte, c Conn) (out []byte, action Action) {
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
			must(err)
			defer conn.Close()
			r := make([]byte, 10)
			_, err = conn.Read(r)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(r))
		}()
		return
	}
	t.c = <-t.conn
	_ = t.c.Wake()
	delay = time.Millisecond * 100
	return
}

func testWakeConn(network, addr string) {
	svr := &testWakeConnServer{network: network, addr: addr, conn: make(chan Conn, 1)}
	logger := zap.NewExample()
	must(Serve(svr, network+"://"+addr, WithTicker(true), WithNumEventLoop(2*runtime.NumCPU()),
		WithLogger(logger.Sugar())))
	_ = logger.Sync()
}

func TestShutdown(t *testing.T) {
	testShutdown("tcp", ":9991")
}

type testShutdownServer struct {
	*EventServer
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
				must(err)
				defer conn.Close()
				_, err = conn.Read([]byte{0})
				if err == nil {
					panic("expected error")
				}
			}()
		}
	} else if int(atomic.LoadInt64(&t.clients)) == t.N {
		action = Shutdown
	}
	t.count++
	delay = time.Second / 20
	return
}

func testShutdown(network, addr string) {
	events := &testShutdownServer{network: network, addr: addr, N: 10}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
	if events.clients != 0 {
		panic("did not call close on all clients")
	}
}

func TestCloseActionError(t *testing.T) {
	testCloseActionError("tcp", ":9992")
}

type testCloseActionErrorServer struct {
	*EventServer
	network, addr string
	action        bool
}

func (t *testCloseActionErrorServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseActionErrorServer) React(frame []byte, c Conn) (out []byte, action Action) {
	out = frame
	action = Close
	return
}

func (t *testCloseActionErrorServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			must(err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(data))
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionError(network, addr string) {
	events := &testCloseActionErrorServer{network: network, addr: addr}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
}

func TestShutdownActionError(t *testing.T) {
	testShutdownActionError("tcp", ":9993")
}

type testShutdownActionErrorServer struct {
	*EventServer
	network, addr string
	action        bool
}

func (t *testShutdownActionErrorServer) React(frame []byte, c Conn) (out []byte, action Action) {
	c.ReadN(-1) // just for test
	out = frame
	action = Shutdown
	return
}

func (t *testShutdownActionErrorServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			must(err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(data))
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionError(network, addr string) {
	events := &testShutdownActionErrorServer{network: network, addr: addr}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
}

func TestCloseActionOnOpen(t *testing.T) {
	testCloseActionOnOpen("tcp", ":9994")
}

type testCloseActionOnOpenServer struct {
	*EventServer
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
			must(err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionOnOpen(network, addr string) {
	events := &testCloseActionOnOpenServer{network: network, addr: addr}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
}

func TestShutdownActionOnOpen(t *testing.T) {
	testShutdownActionOnOpen("tcp", ":9995")
}

type testShutdownActionOnOpenServer struct {
	*EventServer
	network, addr string
	action        bool
}

func (t *testShutdownActionOnOpenServer) OnOpened(c Conn) (out []byte, action Action) {
	action = Shutdown
	return
}

func (t *testShutdownActionOnOpenServer) OnShutdown(s Server) {
	dupFD, err := s.DupFd()
	fmt.Printf("dup fd: %d with error: %v\n", dupFD, err)
}

func (t *testShutdownActionOnOpenServer) Tick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			must(err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionOnOpen(network, addr string) {
	events := &testShutdownActionOnOpenServer{network: network, addr: addr}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
}

func TestUDPShutdown(t *testing.T) {
	testUDPShutdown("udp4", ":9000")
}

type testUDPShutdownServer struct {
	*EventServer
	network string
	addr    string
	tick    bool
}

func (t *testUDPShutdownServer) React(frame []byte, c Conn) (out []byte, action Action) {
	out = frame
	action = Shutdown
	return
}

func (t *testUDPShutdownServer) Tick() (delay time.Duration, action Action) {
	if !t.tick {
		t.tick = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			must(err)
			defer conn.Close()
			data := []byte("Hello World!")
			if _, err = conn.Write(data); err != nil {
				panic(err)
			}
			if _, err = conn.Read(data); err != nil {
				panic(err)
			}
			fmt.Println(string(data))
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testUDPShutdown(network, addr string) {
	svr := &testUDPShutdownServer{network: network, addr: addr}
	must(Serve(svr, network+"://"+addr, WithTicker(true)))
}

func TestCloseConnection(t *testing.T) {
	testCloseConnection("tcp", ":9996")
}

type testCloseConnectionServer struct {
	*EventServer
	network, addr string
	action        bool
}

func (t *testCloseConnectionServer) OnClosed(c Conn, err error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseConnectionServer) React(frame []byte, c Conn) (out []byte, action Action) {
	out = frame
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
			must(err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(data))
			// waiting the server shutdown.
			_, err = conn.Read(data)
			if err == nil {
				panic(err)
			}
		}()
		return
	}
	return
}

func testCloseConnection(network, addr string) {
	events := &testCloseConnectionServer{network: network, addr: addr}
	must(Serve(events, network+"://"+addr, WithTicker(true)))
}

func TestServerOptionsCheck(t *testing.T) {
	if err := Serve(&EventServer{}, "tcp://:3500", WithNumEventLoop(10001), WithLockOSThread(true)); err != errors.ErrTooManyEventLoopThreads {
		t.Fail()
		t.Log("error returned with LockOSThread option")
	}
}

func TestStop(t *testing.T) {
	testStop("tcp", ":9997")
}

type testStopServer struct {
	*EventServer
	network, addr, protoAddr string
	action                   bool
}

func (t *testStopServer) OnClosed(c Conn, err error) (action Action) {
	fmt.Println("closing connection...")
	return
}

func (t *testStopServer) React(frame []byte, c Conn) (out []byte, action Action) {
	out = frame
	return
}

func (t *testStopServer) Tick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			must(err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(data))

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				fmt.Println("stop server...", Stop(ctx, t.protoAddr))
			}()

			// waiting the server shutdown.
			_, err = conn.Read(data)
			if err == nil {
				panic(err)
			}
		}()
		return
	}
	return
}

func testStop(network, addr string) {
	events := &testStopServer{network: network, addr: addr, protoAddr: network + "://" + addr}
	must(Serve(events, events.protoAddr, WithTicker(true)))
}

type testClosedWakeUpServer struct {
	*EventServer
	network, addr, protoAddr string

	readen       chan struct{}
	serverClosed chan struct{}
	clientClosed chan struct{}
}

func (tes *testClosedWakeUpServer) OnInitComplete(_ Server) (action Action) {
	go func() {
		c, err := net.Dial(tes.network, tes.addr)
		if err != nil {
			panic(err)
		}

		_, err = c.Write([]byte("hello"))
		if err != nil {
			panic(err)
		}

		<-tes.readen
		_, err = c.Write([]byte("hello again"))
		if err != nil {
			panic(err)
		}

		must(c.Close())
		close(tes.clientClosed)
		<-tes.serverClosed

		fmt.Println("stop server...", Stop(context.TODO(), tes.protoAddr))
	}()

	return None
}

func (tes *testClosedWakeUpServer) React(_ []byte, conn Conn) ([]byte, Action) {
	if conn.RemoteAddr() == nil {
		panic("react on closed conn")
	}

	select {
	case <-tes.readen:
	default:
		close(tes.readen)
	}

	// actually goroutines here needed only on windows since its async actions
	// rely on an unbuffered channel and since we already into it - this will
	// block forever.
	go func() { must(conn.Wake()) }()
	go func() { must(conn.Close()) }()

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

// Test should not panic when we wake-up server_closed conn.
func TestClosedWakeUp(t *testing.T) {
	events := &testClosedWakeUpServer{
		EventServer: &EventServer{}, network: "tcp", addr: ":8888", protoAddr: "tcp://:8888",
		clientClosed: make(chan struct{}),
		serverClosed: make(chan struct{}),
		readen:       make(chan struct{}),
	}

	must(Serve(events, events.protoAddr))
}
