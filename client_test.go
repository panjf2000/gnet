//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || windows
// +build darwin dragonfly freebsd linux netbsd openbsd windows

package gnet

import (
	"bytes"
	crand "crypto/rand"
	"io"
	"math/rand"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type connHandler struct {
	network string
	rspCh   chan []byte
	data    []byte
}

type clientEvents struct {
	*BuiltinEventEngine
	tester    *testing.T
	svr       *testClient
	packetLen int
}

func (ev *clientEvents) OnBoot(e Engine) Action {
	fd, err := e.Dup()
	require.ErrorIsf(ev.tester, err, errorx.ErrEmptyEngine, "expected error: %v, but got: %v",
		errorx.ErrUnsupportedOp, err)
	assert.EqualValuesf(ev.tester, fd, -1, "expected -1, but got: %d", fd)
	return None
}

var pingMsg = []byte("PING\r\n")

func (ev *clientEvents) OnOpen(Conn) (out []byte, action Action) {
	out = pingMsg
	return
}

func (ev *clientEvents) OnClose(Conn, error) Action {
	if ev.svr != nil {
		if atomic.AddInt32(&ev.svr.clientActive, -1) == 0 {
			return Shutdown
		}
	}
	return None
}

func (ev *clientEvents) OnTraffic(c Conn) (action Action) {
	handler := c.Context().(*connHandler)
	if handler.network == "udp" {
		ev.packetLen = datagramLen
	}
	buf, err := c.Next(-1)
	assert.NoError(ev.tester, err)
	handler.data = append(handler.data, buf...)
	if len(handler.data) < ev.packetLen {
		return
	}
	handler.rspCh <- handler.data
	handler.data = nil
	return
}

func (ev *clientEvents) OnTick() (delay time.Duration, action Action) {
	delay = 200 * time.Millisecond
	return
}

func (ev *clientEvents) OnShutdown(e Engine) {
	fd, err := e.Dup()
	require.ErrorIsf(ev.tester, err, errorx.ErrEmptyEngine, "expected error: %v, but got: %v",
		errorx.ErrUnsupportedOp, err)
	assert.EqualValuesf(ev.tester, fd, -1, "expected -1, but got: %d", fd)
}

func TestClient(t *testing.T) {
	// start an engine
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the engine will be random sizes. 0KB - 1MB.
	// the engine will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll-LT", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{false, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{false, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{false, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{false, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{false, 0, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{false, 0, false, true, true, false, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-ET", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 0, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 0, false, true, true, false, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-ET-chunk", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 1 << 19, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 1 << 19, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 1 << 19, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 1 << 19, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 1 << 19, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 1 << 19, false, true, true, false, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-reuseport-LT", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-reuseport-ET", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9991", &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "tcp", ":9992", &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "udp", ":9991", &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "udp", ":9992", &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet1.sock", &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runClient(t, "unix", "gnet2.sock", &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
	})
}

type testClient struct {
	*BuiltinEventEngine
	client        *Client
	tester        *testing.T
	eng           Engine
	network       string
	addr          string
	multicore     bool
	async         bool
	nclients      int
	started       int32
	connected     int32
	clientActive  int32
	disconnected  int32
	workerPool    *goPool.Pool
	udpReadHeader int32
}

func (s *testClient) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	return
}

func (s *testClient) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(&sync.Once{})
	atomic.AddInt32(&s.connected, 1)
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testClient) OnClose(c Conn, err error) (action Action) {
	if err != nil {
		logging.Debugf("error occurred on closed, %v\n", err)
	}
	if s.network != "udp" {
		require.IsType(s.tester, c.Context(), new(sync.Once), "invalid context")
	}

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
		s.workerPool.Release()
	}

	return
}

func (s *testClient) OnShutdown(Engine) {
	if s.network == "udp" {
		require.EqualValues(s.tester, int32(s.nclients), atomic.LoadInt32(&s.udpReadHeader))
	}
}

func (s *testClient) OnTraffic(c Conn) (action Action) {
	readHeader := func() {
		ping := make([]byte, len(pingMsg))
		n, err := io.ReadFull(c, ping)
		require.NoError(s.tester, err)
		require.EqualValues(s.tester, len(pingMsg), n)
		require.Equal(s.tester, string(pingMsg), string(ping), "bad header")
	}
	v := c.Context()
	if v != nil {
		v.(*sync.Once).Do(readHeader)
	}

	if s.async {
		buf := bbPool.Get()
		_, _ = c.WriteTo(buf)

		if s.network == "tcp" || s.network == "unix" {
			// just for test
			_ = c.InboundBuffered()
			_ = c.OutboundBuffered()
			_, _ = c.Discard(1)
		}
		if v == nil && bytes.Equal(buf.Bytes(), pingMsg) {
			atomic.AddInt32(&s.udpReadHeader, 1)
			buf.Reset()
		}
		_ = s.workerPool.Submit(
			func() {
				if buf.Len() > 0 {
					err := c.AsyncWrite(buf.Bytes(), nil)
					require.NoError(s.tester, err)
				}
			})
		return
	}

	buf, _ := c.Next(-1)
	if v == nil && bytes.Equal(buf, pingMsg) {
		atomic.AddInt32(&s.udpReadHeader, 1)
		buf = nil
	}
	if len(buf) > 0 {
		n, err := c.Write(buf)
		require.NoError(s.tester, err)
		require.EqualValues(s.tester, len(buf), n)
	}
	return
}

func (s *testClient) OnTick() (delay time.Duration, action Action) {
	delay = 100 * time.Millisecond
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			var netConn bool
			if i%2 == 0 {
				netConn = true
			}
			go startGnetClient(s.tester, s.client, s.network, s.addr, s.multicore, s.async, netConn)
		}
	}
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
	return
}

func runClient(t *testing.T, network, addr string, conf *testConf) {
	ts := &testClient{
		tester:     t,
		network:    network,
		addr:       addr,
		multicore:  conf.multicore,
		async:      conf.async,
		nclients:   conf.clients,
		workerPool: goPool.Default(),
	}
	var err error
	clientEV := &clientEvents{tester: t, packetLen: streamLen, svr: ts}
	ts.client, err = NewClient(
		clientEV,
		WithEdgeTriggeredIO(conf.et),
		WithEdgeTriggeredIOChunk(conf.etChunk),
		WithTCPNoDelay(TCPNoDelay),
		WithLockOSThread(true),
		WithTicker(true),
	)
	assert.NoError(t, err)

	err = ts.client.Start()
	assert.NoError(t, err)
	defer ts.client.Stop() //nolint:errcheck

	err = Run(ts,
		network+"://"+addr,
		WithEdgeTriggeredIO(conf.et),
		WithEdgeTriggeredIOChunk(conf.etChunk),
		WithLockOSThread(conf.async),
		WithMulticore(conf.multicore),
		WithReusePort(conf.reuseport),
		WithTicker(true),
		WithTCPKeepAlive(time.Minute*1),
		WithLoadBalancing(conf.lb))
	assert.NoError(t, err)
}

func startGnetClient(t *testing.T, cli *Client, network, addr string, multicore, async, netDial bool) {
	var (
		c   Conn
		err error
	)
	handler := &connHandler{
		network: network,
		rspCh:   make(chan []byte, 1),
	}
	if netDial {
		var netConn net.Conn
		netConn, err = stdDial(network, addr)
		require.NoError(t, err)
		c, err = cli.EnrollContext(netConn, handler)
	} else {
		c, err = cli.DialContext(network, addr, handler)
	}
	require.NoError(t, err)
	defer c.Close()
	err = c.Wake(nil)
	require.NoError(t, err)
	rspCh := handler.rspCh
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	logging.Debugf("test duration: %v", duration)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, streamLen)
		if network == "udp" {
			reqData = reqData[:datagramLen]
		}
		_, err = crand.Read(reqData)
		require.NoError(t, err)
		err = c.AsyncWrite(reqData, nil)
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

type clientEventsForWake struct {
	BuiltinEventEngine
	tester *testing.T
	ch     chan struct{}
}

func (ev *clientEventsForWake) OnBoot(_ Engine) Action {
	ev.ch = make(chan struct{})
	return None
}

func (ev *clientEventsForWake) OnTraffic(c Conn) (action Action) {
	n, err := c.Read(nil)
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, n)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	buf := make([]byte, 10)
	n, err = c.Read(buf)
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, n)
	assert.ErrorIsf(ev.tester, err, io.ErrShortBuffer, "expected error: %v, but got: %v", io.ErrShortBuffer, err)
	buf, err = c.Next(10)
	assert.Nilf(ev.tester, buf, "expected: %v, but got: %v", nil, buf)
	assert.ErrorIsf(ev.tester, err, io.ErrShortBuffer, "expected error: %v, but got: %v", io.ErrShortBuffer, err)
	buf, err = c.Next(-1)
	assert.Emptyf(ev.tester, buf, "expected an empty slice, but got: %v", buf)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	buf, err = c.Peek(10)
	assert.Nilf(ev.tester, buf, "expected: %v, but got: %v", nil, buf)
	assert.ErrorIsf(ev.tester, err, io.ErrShortBuffer, "expected error: %v, but got: %v", io.ErrShortBuffer, err)
	buf, err = c.Peek(-1)
	assert.Emptyf(ev.tester, buf, "expected an empty slice, but got: %v", buf)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	n, err = c.Discard(10)
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, n)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	n, err = c.Discard(-1)
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, n)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	m, err := c.WriteTo(io.Discard)
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, m)
	assert.NoErrorf(ev.tester, err, "expected: %v, but got: %v", nil, err)
	n = c.InboundBuffered()
	assert.Zerof(ev.tester, n, "expected: %v, but got: %v", 0, m)
	<-ev.ch
	return None
}

type serverEventsForWake struct {
	BuiltinEventEngine
	network, addr string
	client        *Client
	clientEV      *clientEventsForWake
	tester        *testing.T
	clients       int32
	started       int32
}

func (ev *serverEventsForWake) OnOpen(_ Conn) ([]byte, Action) {
	atomic.AddInt32(&ev.clients, 1)
	return nil, None
}

func (ev *serverEventsForWake) OnClose(_ Conn, _ error) Action {
	if atomic.AddInt32(&ev.clients, -1) == 0 {
		return Shutdown
	}
	return None
}

func (ev *serverEventsForWake) OnTick() (time.Duration, Action) {
	if atomic.CompareAndSwapInt32(&ev.started, 0, 1) {
		go testConnWakeImmediately(ev.tester, ev.client, ev.clientEV, ev.network, ev.addr)
	}
	return 100 * time.Millisecond, None
}

func testConnWakeImmediately(t *testing.T, client *Client, clientEV *clientEventsForWake, network, addr string) {
	c, err := client.Dial(network, addr)
	assert.NoErrorf(t, err, "failed to dial: %v", err)
	err = c.Wake(nil)
	assert.NoError(t, err)
	err = c.Close()
	assert.NoError(t, err)
	clientEV.ch <- struct{}{}
}

func TestWakeConnImmediately(t *testing.T) {
	currentLogger, currentFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	t.Cleanup(func() {
		logging.SetDefaultLoggerAndFlusher(currentLogger, currentFlusher) // restore
	})

	clientEV := &clientEventsForWake{tester: t}
	logPath := filepath.Join(t.TempDir(), "gnet-test-wake-conn-immediately.log")
	client, err := NewClient(clientEV,
		WithSocketRecvBuffer(4*1024),
		WithSocketSendBuffer(4*1024),
		WithLogPath(logPath),
		WithLogLevel(logging.WarnLevel),
		WithReadBufferCap(512),
		WithWriteBufferCap(512))
	assert.NoError(t, err)
	logging.Cleanup()

	err = client.Start()
	assert.NoError(t, err)
	defer client.Stop() //nolint:errcheck

	serverEV := &serverEventsForWake{tester: t, network: "tcp", addr: ":18888", client: client, clientEV: clientEV}

	err = Run(serverEV, serverEV.network+"://"+serverEV.addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestClientReadOnEOF(t *testing.T) {
	currentLogger, currentFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	t.Cleanup(func() {
		logging.SetDefaultLoggerAndFlusher(currentLogger, currentFlusher) // restore
	})

	ln, err := net.Listen("tcp", "127.0.0.1:9999")
	assert.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			go process(conn)
		}
	}()

	ev := &clientReadOnEOF{
		result: make(chan struct {
			data []byte
			err  error
		}, 1),
		data: []byte("test"),
	}
	cli, err := NewClient(ev,
		WithSocketRecvBuffer(4*1024),
		WithSocketSendBuffer(4*1024),
		WithTCPKeepAlive(time.Minute),
		WithLogger(zap.NewExample().Sugar()),
		WithReadBufferCap(32*1024),
		WithWriteBufferCap(32*1024))
	assert.NoError(t, err)
	defer cli.Stop() //nolint:errcheck

	err = cli.Start()
	assert.NoError(t, err)

	_, err = cli.Dial("tcp", "127.0.0.1:9999")
	assert.NoError(t, err)

	select {
	case res := <-ev.result:
		assert.NoError(t, res.err)
		assert.EqualValuesf(t, ev.data, res.data, "expected: %v, but got: %v", ev.data, res.data)
	case <-time.After(5 * time.Second):
		t.Errorf("timeout waiting for the result")
	}
}

func process(conn net.Conn) {
	defer conn.Close() //noliint:errcheck
	buf := make([]byte, 8)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	_, _ = conn.Write(buf[:n])
	_ = conn.Close()
}

type clientReadOnEOF struct {
	BuiltinEventEngine
	data   []byte
	result chan struct {
		data []byte
		err  error
	}
}

func (clientReadOnEOF) OnBoot(Engine) (action Action) {
	return None
}

func (cli clientReadOnEOF) OnOpen(Conn) (out []byte, action Action) {
	return cli.data, None
}

func (clientReadOnEOF) OnClose(Conn, error) (action Action) {
	return Close
}

func (cli clientReadOnEOF) OnTraffic(c Conn) (action Action) {
	data, err := c.Next(-1)
	cli.result <- struct {
		data []byte
		err  error
	}{data: data, err: err}
	return None
}
