//go:build linux || freebsd || dragonfly || darwin || windows
// +build linux freebsd dragonfly darwin windows

package gnet

import (
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gerr "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type clientEvents struct {
	*BuiltinEventEngine
	tester    *testing.T
	svr       *testClientServer
	packetLen int
	rspChMap  sync.Map
}

func (ev *clientEvents) OnBoot(e Engine) Action {
	fd, err := e.Dup()
	require.ErrorIsf(ev.tester, err, gerr.ErrEmptyEngine, "expected error: %v, but got: %v",
		gerr.ErrUnsupportedOp, err)
	assert.EqualValuesf(ev.tester, fd, -1, "expected -1, but got: %d", fd)
	return None
}

func (ev *clientEvents) OnOpen(c Conn) ([]byte, Action) {
	c.SetContext([]byte{})
	rspCh := make(chan []byte, 1)
	ev.rspChMap.Store(c.LocalAddr().String(), rspCh)
	return nil, None
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
	ctx := c.Context()
	var p []byte
	if ctx != nil {
		p = ctx.([]byte)
	} else { // UDP
		ev.packetLen = 1024
	}
	buf, _ := c.Next(-1)
	p = append(p, buf...)
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

func (ev *clientEvents) OnTick() (delay time.Duration, action Action) {
	delay = 200 * time.Millisecond
	return
}

func (ev *clientEvents) OnShutdown(e Engine) {
	fd, err := e.Dup()
	require.ErrorIsf(ev.tester, err, gerr.ErrEmptyEngine, "expected error: %v, but got: %v",
		gerr.ErrUnsupportedOp, err)
	assert.EqualValuesf(ev.tester, fd, -1, "expected -1, but got: %d", fd)
}

func TestServeWithGnetClient(t *testing.T) {
	// start an engine
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the engine will be random sizes. 0KB - 1MB.
	// the engine will echo back the data.
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
	*BuiltinEventEngine
	client       *Client
	clientEV     *clientEvents
	tester       *testing.T
	eng          Engine
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

func (s *testClientServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	return
}

func (s *testClientServer) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testClientServer) OnClose(c Conn, err error) (action Action) {
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

func (s *testClientServer) OnTraffic(c Conn) (action Action) {
	if s.async {
		buf := bbPool.Get()
		_, _ = c.WriteTo(buf)

		if s.network == "tcp" || s.network == "unix" {
			// just for test
			_ = c.InboundBuffered()
			_ = c.OutboundBuffered()
			_, _ = c.Discard(1)
		}
		_ = s.workerPool.Submit(
			func() {
				_ = c.AsyncWrite(buf.Bytes(), nil)
			})
		return
	}
	buf, _ := c.Next(-1)
	_, _ = c.Write(buf)
	return
}

func (s *testClientServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Second / 5
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			var netConn bool
			if i%2 == 0 {
				netConn = true
			}
			go startGnetClient(s.tester, s.client, s.clientEV, s.network, s.addr, s.multicore, s.async, netConn)
		}
	}
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
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
	ts.clientEV = &clientEvents{tester: t, packetLen: streamLen, svr: ts}
	ts.client, err = NewClient(
		ts.clientEV,
		WithLogLevel(logging.DebugLevel),
		WithLockOSThread(true),
		WithTicker(true),
	)
	assert.NoError(t, err)

	err = ts.client.Start()
	assert.NoError(t, err)
	defer ts.client.Stop() //nolint:errcheck

	err = Run(ts,
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

func startGnetClient(t *testing.T, cli *Client, ev *clientEvents, network, addr string, multicore, async, netDial bool) {
	rand.Seed(time.Now().UnixNano())
	var (
		c   Conn
		err error
	)
	if netDial {
		var netConn net.Conn
		netConn, err = NetDial(network, addr)
		require.NoError(t, err)
		c, err = cli.Enroll(netConn)
	} else {
		c, err = cli.Dial(network, addr)
	}
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
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	t.Logf("test duration: %dms", duration/time.Millisecond)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, streamLen)
		if network == "udp" {
			reqData = reqData[:1024]
		}
		_, err = rand.Read(reqData)
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
