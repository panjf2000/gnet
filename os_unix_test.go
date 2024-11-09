//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package gnet

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

var (
	SysClose = unix.Close
	stdDial  = net.Dial
)

// NOTE: TestServeMulticast can fail with "write: no buffer space available" on Wi-Fi interface.
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := net.Dial("udp", s.addr)
	require.NoError(s.t, err)
	defer c.Close()
	ch := make(chan []byte, 10000)
	s.mcast.Store(c.LocalAddr().String(), ch)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	logging.Debugf("test duration: %v", duration)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, 1024)
		_, err = crand.Read(reqData)
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

func detectLinuxEthernetInterfaceName() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	// Traditionally, network interfaces were named as eth0, eth1, etc., for Ethernet interfaces.
	// However, with the introduction of predictable network interface names. Meanwhile, modern
	// convention commonly uses patterns like eno[1-N], ens[1-N], enp<PCI slot>s<card index no>, etc.,
	// for Ethernet interfaces.
	// Check out https://www.thomas-krenn.com/en/wiki/Predictable_Network_Interface_Names and
	// https://en.wikipedia.org/wiki/Consistent_Network_Device_Naming for more details.
	regex := regexp.MustCompile(`e(no|ns|np|th)\d+s*\d*$`)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagRunning == 0 {
			continue
		}
		if regex.MatchString(iface.Name) {
			return iface.Name, nil
		}
	}
	return "", errors.New("no Ethernet interface found")
}

func getInterfaceIP(ifname string, ipv4 bool) (net.IP, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, err
	}
	// Get all unicast addresses for this interface
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	// Loop through the addresses and find the first IPv4 address
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		// Check if the IP is IPv4.
		if ip != nil && (ip.To4() != nil) == ipv4 {
			return ip, nil
		}
	}
	return nil, errors.New("no valid IP address found")
}

type testBindToDeviceServer[T interface{ *net.TCPAddr | *net.UDPAddr }] struct {
	BuiltinEventEngine
	tester          *testing.T
	data            []byte
	packets         atomic.Int32
	expectedPackets int32
	network         string
	loopBackAddr    T
	eth0Addr        T
	broadcastAddr   T
}

func netDial[T *net.TCPAddr | *net.UDPAddr](network string, a T) (net.Conn, error) {
	addr := any(a)
	switch v := addr.(type) {
	case *net.TCPAddr:
		return net.DialTCP(network, nil, v)
	case *net.UDPAddr:
		return net.DialUDP(network, nil, v)
	default:
		return nil, errors.New("unsupported address type")
	}
}

func (s *testBindToDeviceServer[T]) OnTraffic(c Conn) (action Action) {
	b, err := c.Next(-1)
	assert.NoError(s.tester, err)
	assert.EqualValues(s.tester, s.data, b)
	_, err = c.Write(b)
	assert.NoError(s.tester, err)
	s.packets.Add(1)
	return
}

func (s *testBindToDeviceServer[T]) OnShutdown(_ Engine) {
	assert.EqualValues(s.tester, s.expectedPackets, s.packets.Load())
}

func (s *testBindToDeviceServer[T]) OnTick() (delay time.Duration, action Action) {
	// Send a packet to the loopback interface, it should never make its way to the server
	// because we've bound the server to eth0.
	c, err := netDial(s.network, s.loopBackAddr)
	if strings.HasPrefix(s.network, "tcp") {
		assert.ErrorContains(s.tester, err, "connection refused")
	} else {
		assert.NoError(s.tester, err)
		defer c.Close()
		_, err = c.Write(s.data)
		assert.NoError(s.tester, err)
	}

	if s.broadcastAddr != nil {
		// Send a packet to the broadcast address, it should reach the server.
		c6, err := netDial(s.network, s.broadcastAddr)
		assert.NoError(s.tester, err)
		defer c6.Close()
		_, err = c6.Write(s.data)
		assert.NoError(s.tester, err)
	}

	// Send a packet to the eth0 interface, it should reach the server.
	c4, err := netDial(s.network, s.eth0Addr)
	assert.NoError(s.tester, err)
	defer c4.Close()
	_, err = c4.Write(s.data)
	assert.NoError(s.tester, err)
	buf := make([]byte, len(s.data))
	_, err = c4.Read(buf)
	assert.NoError(s.tester, err)
	assert.EqualValues(s.tester, s.data, buf, len(s.data), len(buf))

	return time.Second, Shutdown
}

func TestBindToDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		err := Run(&testBindToDeviceServer[*net.UDPAddr]{}, "tcp://:9999", WithBindToDevice("eth0"))
		assert.ErrorIs(t, err, errorx.ErrUnsupportedOp)
		return
	}

	lp, err := findLoopbackInterface()
	assert.NoError(t, err)
	dev, err := detectLinuxEthernetInterfaceName()
	assert.NoErrorf(t, err, "no testable Ethernet interface found")
	t.Logf("detected Ethernet interface: %s", dev)
	data := []byte("hello")
	t.Run("IPv4", func(t *testing.T) {
		ip, err := getInterfaceIP(dev, true)
		assert.NoError(t, err)
		t.Run("TCP", func(t *testing.T) {
			ts := &testBindToDeviceServer[*net.TCPAddr]{
				tester:          t,
				data:            data,
				expectedPackets: 1,
				network:         "tcp",
				loopBackAddr:    &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999, Zone: ""},
				eth0Addr:        &net.TCPAddr{IP: ip, Port: 9999, Zone: ""},
			}
			require.NoError(t, err)
			err = Run(ts, "tcp://0.0.0.0:9999",
				WithTicker(true),
				WithBindToDevice(dev))
			assert.NoError(t, err)
		})
		t.Run("UDP", func(t *testing.T) {
			ts := &testBindToDeviceServer[*net.UDPAddr]{
				tester:          t,
				data:            data,
				expectedPackets: 2,
				network:         "udp",
				loopBackAddr:    &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999, Zone: ""},
				eth0Addr:        &net.UDPAddr{IP: ip, Port: 9999, Zone: ""},
				broadcastAddr:   &net.UDPAddr{IP: net.IPv4bcast, Port: 9999, Zone: ""},
			}
			require.NoError(t, err)
			err = Run(ts, "udp://0.0.0.0:9999",
				WithTicker(true),
				WithBindToDevice(dev))
			assert.NoError(t, err)
		})
	})
	t.Run("IPv6", func(t *testing.T) {
		t.Run("TCP", func(t *testing.T) {
			ip, err := getInterfaceIP(dev, false)
			assert.NoError(t, err)
			ts := &testBindToDeviceServer[*net.TCPAddr]{
				tester:          t,
				data:            data,
				expectedPackets: 1,
				network:         "tcp6",
				loopBackAddr:    &net.TCPAddr{IP: net.IPv6loopback, Port: 9999, Zone: lp.Name},
				eth0Addr:        &net.TCPAddr{IP: ip, Port: 9999, Zone: dev},
			}
			require.NoError(t, err)
			err = Run(ts, "tcp6://[::]:9999",
				WithTicker(true),
				WithBindToDevice(dev))
			assert.NoError(t, err)
		})
		t.Run("UDP", func(t *testing.T) {
			ip, err := getInterfaceIP(dev, false)
			assert.NoError(t, err)
			ts := &testBindToDeviceServer[*net.UDPAddr]{
				tester:          t,
				data:            data,
				expectedPackets: 2,
				network:         "udp6",
				loopBackAddr:    &net.UDPAddr{IP: net.IPv6loopback, Port: 9999, Zone: lp.Name},
				eth0Addr:        &net.UDPAddr{IP: ip, Port: 9999, Zone: dev},
				broadcastAddr:   &net.UDPAddr{IP: net.IPv6linklocalallnodes, Port: 9999, Zone: dev},
			}
			require.NoError(t, err)
			err = Run(ts, "udp6://[::]:9999",
				WithTicker(true),
				WithBindToDevice(dev))
			assert.NoError(t, err)
		})
	})
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
