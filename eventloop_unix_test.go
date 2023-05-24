//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package gnet

import (
	"context"
	"net"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/assert"
)

func (lb *roundRobinLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

func (lb *leastConnectionsLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

func (lb *sourceAddrHashLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

func registerInitConn(el *eventloop) {
	for i := 0; i < int(atomic.LoadInt32(&nowEventLoopInitConn)); i++ {
		c := newTCPConn(i, el, &unix.SockaddrInet4{}, &net.TCPAddr{}, &net.TCPAddr{})
		el.connections.addConn(c, el.idx)
	}
}

// nowEventLoopInitConn initializes the number of conn fake data, must be set to 0 after use.
var (
	nowEventLoopInitConn int32
	testBigGC            = false
)

func BenchmarkGC4El100k(b *testing.B) {
	oldGc := debug.SetGCPercent(-1)

	ts1 := benchServeGC(b, "tcp", ":9001", true, 4, 100000)
	b.Run("Run-4-eventloop-100000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
		}
	})
	_ = ts1.eng.Stop(context.Background())

	debug.SetGCPercent(oldGc)
}

func BenchmarkGC4El200k(b *testing.B) {
	oldGc := debug.SetGCPercent(-1)

	ts1 := benchServeGC(b, "tcp", ":9001", true, 4, 200000)
	b.Run("Run-4-eventloop-200000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
		}
	})
	_ = ts1.eng.Stop(context.Background())

	debug.SetGCPercent(oldGc)
}

func BenchmarkGC4El500k(b *testing.B) {
	oldGc := debug.SetGCPercent(-1)

	ts1 := benchServeGC(b, "tcp", ":9001", true, 4, 500000)
	b.Run("Run-4-eventloop-500000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
		}
	})
	_ = ts1.eng.Stop(context.Background())

	debug.SetGCPercent(oldGc)
}

func benchServeGC(b *testing.B, network, addr string, async bool, elNum int, initConnCount int32) *benchmarkServerGC {
	ts := &benchmarkServerGC{
		tester:        b,
		network:       network,
		addr:          addr,
		async:         async,
		elNum:         elNum,
		initOk:        make(chan struct{}),
		initConnCount: initConnCount,
	}

	nowEventLoopInitConn = initConnCount
	go func() {
		err := Run(ts,
			network+"://"+addr,
			WithLockOSThread(async),
			WithNumEventLoop(elNum),
			WithTCPKeepAlive(time.Minute*1),
			WithTCPNoDelay(TCPDelay))
		assert.NoError(b, err)
		nowEventLoopInitConn = 0
	}()
	<-ts.initOk
	return ts
}

type benchmarkServerGC struct {
	*BuiltinEventEngine
	tester        *testing.B
	eng           Engine
	network       string
	addr          string
	async         bool
	elNum         int
	initConnCount int32
	initOk        chan struct{}
}

func (s *benchmarkServerGC) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	go func() {
		for {
			if s.eng.eng.eventLoops.len() == s.elNum && s.eng.CountConnections() == s.elNum*int(s.initConnCount) {
				break
			}
			time.Sleep(time.Millisecond)
		}
		close(s.initOk)
	}()
	return
}

// TestServeGC generate fake data asynchronously, if you need to test, manually open the comment.
func TestServeGC(t *testing.T) {
	t.Run("gc-loop", func(t *testing.T) {
		t.Run("1-loop-10000", func(t *testing.T) {
			if testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 1, 10000)
		})
		t.Run("1-loop-100000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 1, 100000)
		})
		t.Run("1-loop-1000000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 1, 1000000)
		})
		t.Run("2-loop-10000", func(t *testing.T) {
			if testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 2, 10000)
		})
		t.Run("2-loop-100000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 2, 100000)
		})
		t.Run("2-loop-1000000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 2, 1000000)
		})
		t.Run("4-loop-10000", func(t *testing.T) {
			if testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 4, 10000)
		})
		t.Run("4-loop-100000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 4, 100000)
		})
		t.Run("4-loop-1000000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 4, 1000000)
		})
		t.Run("16-loop-10000", func(t *testing.T) {
			if testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 16, 10000)
		})
		t.Run("16-loop-100000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 16, 100000)
		})
		t.Run("16-loop-1000000", func(t *testing.T) {
			if !testBigGC {
				t.Skipf("Skip when testBigGC=%t", testBigGC)
			}
			testServeGC(t, "tcp", ":9000", true, true, 16, 1000000)
		})
	})
}

func testServeGC(t *testing.T, network, addr string, multicore, async bool, elNum int, initConnCount int32) {
	ts := &testServerGC{
		tester:    t,
		network:   network,
		addr:      addr,
		multicore: multicore,
		async:     async,
		elNum:     elNum,
	}

	nowEventLoopInitConn = initConnCount

	err := Run(ts,
		network+"://"+addr,
		WithLockOSThread(async),
		WithMulticore(multicore),
		WithNumEventLoop(elNum),
		WithTCPKeepAlive(time.Minute*1),
		WithTCPNoDelay(TCPDelay))
	assert.NoError(t, err)
	nowEventLoopInitConn = 0
}

type testServerGC struct {
	*BuiltinEventEngine
	tester    *testing.T
	eng       Engine
	network   string
	addr      string
	multicore bool
	async     bool
	elNum     int
}

func (s *testServerGC) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	gcSecs := 5
	if testBigGC {
		gcSecs = 10
	}
	go s.GC(gcSecs)

	return
}

func (s *testServerGC) GC(secs int) {
	defer func() {
		_ = s.eng.Stop(context.Background())
		runtime.GC()
	}()
	var gcAllTime, gcAllCount time.Duration
	gcStart := time.Now()
	for range time.Tick(time.Second) {
		gcAllCount++
		now := time.Now()
		runtime.GC()
		gcTime := time.Since(now)
		gcAllTime += gcTime
		s.tester.Log(s.tester.Name(), s.network, " server gc:", gcTime, ", average gc time:", gcAllTime/gcAllCount)
		if time.Since(gcStart) >= time.Second*time.Duration(secs) {
			break
		}
	}
}
