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
	"go.uber.org/zap/zapcore"

	"github.com/panjf2000/gnet/pool/goroutine"
)

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
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, false, 10, false, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, false, 10, false, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, true, 10, false, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, true, 10, false, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, false, 10, true, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, false, 10, true, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9993", false, true, 10, true, NewFixedLengthFrameCodec(64))
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
				testCodecServeWithGnetClient(t, "tcp", ":9997", true, true, 10, true, NewFixedLengthFrameCodec(64))
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
	tester       *testing.T
	network      string
	addr         string
	multicore    bool
	async        bool
	started      int32
	connected    int32
	disconnected int32
	codec        ICodec
	workerPool   *goroutine.Pool
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
		atomic.LoadInt32(&s.disconnected) == 1 {
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
	if atomic.LoadInt32(&s.started) == 0 {
		go startCodecGnetClient(s.tester, s.client, s.clientEV, s.network, s.addr, s.multicore, s.async, s.codec)
		atomic.StoreInt32(&s.started, 1)
	}
	delay = time.Second / 5
	return
}

type clientEvents struct {
	*EventServer
	rspChMap sync.Map
}

func (cli *clientEvents) OnOpened(c Conn) ([]byte, Action) {
	rspCh := make(chan []byte, 128)
	cli.rspChMap.Store(c.LocalAddr().String(), rspCh)
	return nil, None
}

func (cli *clientEvents) React(packet []byte, c Conn) (out []byte, action Action) {
	v, _ := cli.rspChMap.Load(c.LocalAddr().String())
	rspCh := v.(chan []byte)
	rspCh <- append([]byte{}, packet...)
	return
}

var (
	m               = 0
	allFieldLengths = []int{1, 2, 3, 4, 8}
)

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
		fieldLength := allFieldLengths[m]
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
	m++
	if m > 4 {
		m = 0
	}
	ts := &testCodecClientServer{
		tester: t, network: network, addr: addr, multicore: multicore, async: async,
		codec: codec, workerPool: goroutine.Default(),
	}
	ts.clientEV = &clientEvents{}
	ts.client, err = NewClient(ts.clientEV, WithLogLevel(zapcore.DebugLevel), WithCodec(codec))
	err = ts.client.Start()
	assert.NoError(t, err)
	err = Serve(
		ts,
		network+"://"+addr,
		WithMulticore(multicore),
		WithTicker(true),
		WithLogLevel(zapcore.DebugLevel),
		WithTCPKeepAlive(
			time.Minute*5,
		),
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
		v interface{}
	)
	for {
		v, ok = ev.rspChMap.Load(c.LocalAddr().String())
		if ok {
			break
		}
	}
	rspCh := v.(chan []byte)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		reqData := []byte(strings.Repeat("abcd1234", 8)) // 64 bytes
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
