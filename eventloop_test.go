package gnet

import (
	"context"
	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"time"
)

func TestServeGC(t *testing.T) {
	//wg := &sync.WaitGroup{}
	t.Run("gc-loop", func(t *testing.T) {
		t.Run("1-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 1, 10000)
		})
		t.Run("1-loop-100000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 1, 100000)
		})
		t.Run("1-loop-1000000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 1, 1000000)
		})
		t.Run("2-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 2, 10000)
		})
		t.Run("2-loop-100000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 2, 100000)
		})
		t.Run("2-loop-1000000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 2, 1000000)
		})
		t.Run("4-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 4, 10000)
		})
		t.Run("4-loop-100000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 4, 100000)
		})
		t.Run("4-loop-1000000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 4, 1000000)
		})
		t.Run("16-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 16, 10000)
		})
		t.Run("16-loop-100000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 16, 100000)
		})
		t.Run("16-loop-1000000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 16, 1000000)
		})
	})

	//wg.Wait()
}

func testServeGC(t *testing.T, network, addr string, multicore, async bool, elNum, initConnCount int) {
	ts := &testServerGC{
		tester:        t,
		network:       network,
		addr:          addr,
		multicore:     multicore,
		async:         async,
		workerPool:    goPool.Default(),
		elNum:         elNum,
		initConnCount: initConnCount,
	}

	err := Run(ts,
		network+"://"+addr,
		WithLockOSThread(async),
		WithMulticore(multicore),
		WithNumEventLoop(elNum),
		WithTCPKeepAlive(time.Minute*1),
		WithTCPNoDelay(TCPDelay))
	assert.NoError(t, err)
}

type testServerGC struct {
	*BuiltinEventEngine
	tester        *testing.T
	eng           Engine
	network       string
	addr          string
	multicore     bool
	async         bool
	writev        bool
	nclients      int
	started       int32
	connected     int32
	clientActive  int32
	disconnected  int32
	workerPool    *goPool.Pool
	elNum         int
	initConnCount int
}

func (s *testServerGC) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	addr := eng.eng.ln.addr
	go func() {
		defer eng.Stop(context.Background())
		for eng.eng.lb.len() == 0 {
			time.Sleep(time.Second)
		}
		for elIdx := 0; elIdx < s.elNum; elIdx++ {
			el := eng.eng.lb.index(elIdx)
			for i := 0; i < s.initConnCount; i++ {
				c := newTCPConn(i, el, nil, addr, addr)
				el.storeConn(c)
			}
		}

		d := time.Duration(0)
		dc := d
		start := time.Now()
		for range time.Tick(time.Second * 2) {
			dc++
			nn := time.Now()
			runtime.GC()
			td := time.Since(nn)
			d += td
			s.tester.Log(s.tester.Name(), s.network, " server gc:", td, ", average gc time: ", d/dc, ",conn count:", eng.CountConnections())
			if time.Since(start) > time.Second*10 {
				break
			}
		}
	}()

	return
}

func (s *testServerGC) OnOpen(c Conn) (out []byte, action Action) {

	return
}

func (s *testServerGC) OnClose(c Conn, err error) (action Action) {

	return
}

func (s *testServerGC) OnTraffic(c Conn) (action Action) {
	return
}

func (s *testServerGC) OnTick() (delay time.Duration, action Action) {
	return
}
