//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var (
	SysClose = unix.Close
	NetDial  = net.Dial
)

// NOTE: TestServeMulticast can fail with "write: no buffer space available" on wifi interface.
func TestServeMulticast(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		// 224.0.0.169 is an unassigned address from the Local Network Control Block
		// https://www.iana.org/assignments/multicast-addresses/multicast-addresses.xhtml#multicast-addresses-1
		t.Run("udp-multicast", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", false, false, -1, 10)
		})
		t.Run("udp-multicast-reuseport", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", true, false, -1, 10)
		})
		t.Run("udp-multicast-reuseaddr", func(t *testing.T) {
			testMulticast(t, "224.0.0.169:9991", false, true, -1, 10)
		})
	})
	t.Run("IPv6", func(t *testing.T) {
		iface, err := findLoopbackInterface()
		require.NoError(t, err)
		if iface.Flags&net.FlagMulticast != net.FlagMulticast {
			t.Skip("multicast is not supported on loopback interface")
		}
		// ff02::3 is an unassigned address from Link-Local Scope Multicast Addresses
		// https://www.iana.org/assignments/ipv6-multicast-addresses/ipv6-multicast-addresses.xhtml#link-local
		t.Run("udp-multicast", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), false, false, iface.Index, 10)
		})
		t.Run("udp-multicast-reuseport", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), true, false, iface.Index, 10)
		})
		t.Run("udp-multicast-reuseaddr", func(t *testing.T) {
			testMulticast(t, fmt.Sprintf("[ff02::3%%%s]:9991", iface.Name), false, true, iface.Index, 10)
		})
	})
}

func findLoopbackInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == net.FlagLoopback {
			return &iface, nil
		}
	}
	return nil, errors.New("no loopback interface")
}

func testMulticast(t *testing.T, addr string, reuseport, reuseaddr bool, index, nclients int) {
	ts := &testMcastServer{
		t:        t,
		addr:     addr,
		nclients: nclients,
	}
	options := []Option{
		WithReuseAddr(reuseaddr),
		WithReusePort(reuseport),
		WithSocketRecvBuffer(2 * nclients * 1024), // enough space to receive messages from nclients to eliminate dropped packets
		WithTicker(true),
	}
	if index != -1 {
		options = append(options, WithMulticastInterfaceIndex(index))
	}
	err := Run(ts, "udp://"+addr, options...)
	assert.NoError(t, err)
}

type testMcastServer struct {
	*BuiltinEventEngine
	t        *testing.T
	mcast    sync.Map
	addr     string
	nclients int
	started  int32
	active   int32
}

func (s *testMcastServer) startMcastClient() {
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := net.Dial("udp", s.addr)
	require.NoError(s.t, err)
	defer c.Close()
	ch := make(chan []byte, 10000)
	s.mcast.Store(c.LocalAddr().String(), ch)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	s.t.Logf("test duration: %dms", duration/time.Millisecond)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, 1024)
		_, err = rand.Read(reqData)
		require.NoError(s.t, err)
		_, err = c.Write(reqData)
		require.NoError(s.t, err)
		// Workaround for MacOS "write: no buffer space available" error messages
		// https://developer.apple.com/forums/thread/42334
		time.Sleep(time.Millisecond)
		select {
		case respData := <-ch:
			require.Equalf(s.t, reqData, respData, "response mismatch, length of bytes: %d vs %d", len(reqData), len(respData))
		case <-ctx.Done():
			require.Fail(s.t, "timeout receiving message")
			return
		}
	}
}

func (s *testMcastServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Next(-1)
	b := make([]byte, len(buf))
	copy(b, buf)
	ch, ok := s.mcast.Load(c.RemoteAddr().String())
	require.True(s.t, ok)
	ch.(chan []byte) <- b
	return
}

func (s *testMcastServer) OnTick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.active, 1)
			go func() {
				defer atomic.AddInt32(&s.active, -1)
				s.startMcastClient()
			}()
		}
	}
	if atomic.LoadInt32(&s.active) == 0 {
		action = Shutdown
		return
	}
	delay = time.Second / 5
	return
}

type testMulticastBindServer struct {
	*BuiltinEventEngine
}

func (t *testMulticastBindServer) OnTick() (delay time.Duration, action Action) {
	action = Shutdown
	return
}

func TestMulticastBindIPv4(t *testing.T) {
	ts := &testMulticastBindServer{}
	iface, err := findLoopbackInterface()
	require.NoError(t, err)
	err = Run(ts, "udp://224.0.0.169:9991",
		WithMulticastInterfaceIndex(iface.Index),
		WithTicker(true))
	assert.NoError(t, err)
}

func TestMulticastBindIPv6(t *testing.T) {
	ts := &testMulticastBindServer{}
	iface, err := findLoopbackInterface()
	require.NoError(t, err)
	err = Run(ts, fmt.Sprintf("udp://[ff02::3%%%s]:9991", iface.Name),
		WithMulticastInterfaceIndex(iface.Index),
		WithTicker(true))
	assert.NoError(t, err)
}

/*
func TestEngineAsyncWrite(t *testing.T) {
	t.Run("tcp", func(t *testing.T) {
		t.Run("1-loop", func(t *testing.T) {
			testEngineAsyncWrite(t, "tcp", ":18888", false, false, 10, LeastConnections)
		})
		t.Run("N-loop", func(t *testing.T) {
			testEngineAsyncWrite(t, "tcp", ":28888", true, true, 10, RoundRobin)
		})
	})
	t.Run("unix", func(t *testing.T) {
		t.Run("1-loop", func(t *testing.T) {
			testEngineAsyncWrite(t, "unix", ":18888", false, false, 10, LeastConnections)
		})
		t.Run("N-loop", func(t *testing.T) {
			testEngineAsyncWrite(t, "unix", ":28888", true, true, 10, RoundRobin)
		})
	})
}

type testEngineAsyncWriteServer struct {
	*BuiltinEventEngine
	tester       *testing.T
	eng          Engine
	network      string
	addr         string
	multicore    bool
	writev       bool
	nclients     int
	started      int32
	connected    int32
	clientActive int32
	disconnected int32
	workerPool   *goPool.Pool
}

func (s *testEngineAsyncWriteServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	return
}

func (s *testEngineAsyncWriteServer) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	out = []byte("sweetness\r\n")
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testEngineAsyncWriteServer) OnClose(c Conn, err error) (action Action) {
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

func (s *testEngineAsyncWriteServer) OnTraffic(c Conn) (action Action) {
	gFD := c.Gfd()

	buf := bbPool.Get()
	_, _ = c.WriteTo(buf)

	// just for test
	_ = c.InboundBuffered()
	_ = c.OutboundBuffered()
	_, _ = c.Discard(1)

	_ = s.workerPool.Submit(
		func() {
			if s.writev {
				mid := buf.Len() / 2
				bs := make([][]byte, 2)
				bs[0] = buf.B[:mid]
				bs[1] = buf.B[mid:]
				_ = s.eng.AsyncWritev(gFD, bs, func(c Conn, err error) error {
					if c.RemoteAddr() != nil {
						logging.Debugf("conn=%s done writev: %v", c.RemoteAddr().String(), err)
					}
					bbPool.Put(buf)
					return nil
				})
			} else {
				_ = s.eng.AsyncWrite(gFD, buf.Bytes(), func(c Conn, err error) error {
					if c.RemoteAddr() != nil {
						logging.Debugf("conn=%s done write: %v", c.RemoteAddr().String(), err)
					}
					bbPool.Put(buf)
					return nil
				})
			}
		})
	return
}

func (s *testEngineAsyncWriteServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Second / 5
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			atomic.AddInt32(&s.clientActive, 1)
			go func() {
				initClient(s.tester, s.network, s.addr, s.multicore)
				atomic.AddInt32(&s.clientActive, -1)
			}()
		}
	}
	if s.network == "udp" && atomic.LoadInt32(&s.clientActive) == 0 {
		action = Shutdown
		return
	}
	return
}

func testEngineAsyncWrite(t *testing.T, network, addr string, multicore, writev bool, nclients int, lb LoadBalancing) {
	ts := &testEngineAsyncWriteServer{
		tester:     t,
		network:    network,
		addr:       addr,
		multicore:  multicore,
		writev:     writev,
		nclients:   nclients,
		workerPool: goPool.Default(),
	}
	err := Run(ts,
		network+"://"+addr,
		WithMulticore(multicore),
		WithTicker(true),
		WithLoadBalancing(lb))
	assert.NoError(t, err)
}

func initClient(t *testing.T, network, addr string, multicore bool) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	require.NoError(t, err)
	require.Equal(t, string(msg), "sweetness\r\n", "bad header")
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	t.Logf("test duration: %dms", duration/time.Millisecond)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, streamLen)
		_, err = rand.Read(reqData)
		require.NoError(t, err)
		_, err = c.Write(reqData)
		require.NoError(t, err)
		respData := make([]byte, len(reqData))
		_, err = io.ReadFull(rd, respData)
		require.NoError(t, err)
		require.Equalf(
			t,
			len(reqData),
			len(respData),
			"response mismatch with protocol:%s, multi-core:%t, length of bytes: %d vs %d",
			network,
			multicore,
			len(reqData),
			len(respData),
		)
	}
}

func TestEngineWakeConn(t *testing.T) {
	testEngineWakeConn(t, "tcp", ":9990")
}

type testEngineWakeConnServer struct {
	*BuiltinEventEngine
	tester  *testing.T
	eng     Engine
	network string
	addr    string
	gFD     chan gfd.GFD
	wake    bool
}

func (t *testEngineWakeConnServer) OnBoot(eng Engine) (action Action) {
	t.eng = eng
	return
}

func (t *testEngineWakeConnServer) OnOpen(c Conn) (out []byte, action Action) {
	t.gFD <- c.Gfd()
	return
}

func (t *testEngineWakeConnServer) OnClose(Conn, error) (action Action) {
	action = Shutdown
	return
}

func (t *testEngineWakeConnServer) OnTraffic(c Conn) (action Action) {
	_, _ = c.Write([]byte("Waking up."))
	action = -1
	return
}

func (t *testEngineWakeConnServer) OnTick() (delay time.Duration, action Action) {
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
	gFD := <-t.gFD
	_ = t.eng.Wake(gFD, func(c Conn, err error) error {
		logging.Debugf("conn=%s done wake: %v", c.RemoteAddr().String(), err)
		return nil
	})
	delay = time.Millisecond * 100
	return
}

func testEngineWakeConn(t *testing.T, network, addr string) {
	svr := &testEngineWakeConnServer{tester: t, network: network, addr: addr, gFD: make(chan gfd.GFD, 1)}
	logger := zap.NewExample()
	err := Run(svr, network+"://"+addr,
		WithTicker(true),
		WithNumEventLoop(2*runtime.NumCPU()),
		WithLogger(logger.Sugar()),
		WithSocketRecvBuffer(4*1024),
		WithSocketSendBuffer(4*1024),
		WithReadBufferCap(2000),
		WithWriteBufferCap(2000))
	assert.NoError(t, err)
	_ = logger.Sync()
}

// Test should not panic when we wake-up server_closed conn.
func TestEngineClosedWakeUp(t *testing.T) {
	events := &testEngineClosedWakeUpServer{
		tester:             t,
		BuiltinEventEngine: &BuiltinEventEngine{}, network: "tcp", addr: ":9999", protoAddr: "tcp://:9999",
		clientClosed: make(chan struct{}),
		serverClosed: make(chan struct{}),
		wakeup:       make(chan struct{}),
	}

	err := Run(events, events.protoAddr)
	assert.NoError(t, err)
}

type testEngineClosedWakeUpServer struct {
	*BuiltinEventEngine
	tester                   *testing.T
	network, addr, protoAddr string

	eng Engine

	wakeup       chan struct{}
	serverClosed chan struct{}
	clientClosed chan struct{}
}

func (s *testEngineClosedWakeUpServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	go func() {
		c, err := net.Dial(s.network, s.addr)
		require.NoError(s.tester, err)

		_, err = c.Write([]byte("hello"))
		require.NoError(s.tester, err)

		<-s.wakeup
		_, err = c.Write([]byte("hello again"))
		require.NoError(s.tester, err)

		close(s.clientClosed)
		<-s.serverClosed

		logging.Debugf("stop engine...", Stop(context.TODO(), s.protoAddr))
	}()

	return None
}

func (s *testEngineClosedWakeUpServer) OnTraffic(c Conn) Action {
	assert.NotNil(s.tester, c.RemoteAddr())

	select {
	case <-s.wakeup:
	default:
		close(s.wakeup)
	}

	fd := c.Gfd()

	go func() { require.NoError(s.tester, c.Wake(nil)) }()
	go s.eng.Close(fd, nil)

	<-s.clientClosed

	_, _ = c.Write([]byte("answer"))
	return None
}

func (s *testEngineClosedWakeUpServer) OnClose(Conn, error) (action Action) {
	select {
	case <-s.serverClosed:
	default:
		close(s.serverClosed)
	}
	return
}
*/
