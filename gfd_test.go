package gnet

import (
	"context"
	"math"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/panjf2000/gnet/v2/pkg/gfd"
)

func TestServeGFD(t *testing.T) {
	t.Run("GFD-1-loop-10000", func(t *testing.T) {
		testServeGFD(t, "tcp", ":9000", true, true, 1, 10000)
	})
	t.Run("GFD-2-loop-10000", func(t *testing.T) {
		testServeGFD(t, "tcp", ":9000", true, true, 2, 10000)
	})
	t.Run("GFD-4-loop-10000", func(t *testing.T) {
		testServeGFD(t, "tcp", ":9000", true, true, 4, 10000)
	})
	t.Run("GFD-8-loop-10000", func(t *testing.T) {
		testServeGFD(t, "tcp", ":9000", true, true, 8, 10000)
	})
	t.Run("GFD-16-loop-10000", func(t *testing.T) {
		testServeGFD(t, "tcp", ":9000", true, true, 16, 10000)
	})
}

func testServeGFD(t *testing.T, network, addr string, multicore, async bool, elNum int, initConnCount int32) {
	ts := &testServerGFD{
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
		WithTicker(true),
		WithNumEventLoop(elNum),
		WithTCPKeepAlive(time.Minute*1),
		WithTCPNoDelay(TCPDelay))
	assert.NoError(t, err)
}

type testServerGFD struct {
	*BuiltinEventEngine
	tester    *testing.T
	eng       Engine
	network   string
	addr      string
	multicore bool
	async     bool
	elNum     int
	started   int32
	client    []net.Conn
	real      sync.Map
	fail      sync.Map
}

func (s *testServerGFD) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	return
}

func (s *testServerGFD) OnOpen(c Conn) (out []byte, action Action) {
	s.real.Store(c.Fd(), c.Gfd())
	return
}

func (s *testServerGFD) OnClose(c Conn, _ error) (action Action) {
	s.fail.Store(c.Fd(), c.Gfd())
	s.real.Delete(c.Fd())
	return
}

func (s *testServerGFD) OnTick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		// 建立客户端连接
		for i := 0; i < 100; i++ {
			c, err := net.Dial(s.network, s.addr)
			assert.NoError(s.tester, err)
			s.client = append(s.client, c)
		}
		return time.Millisecond * 10, None
	}
	// 检测正常数据
	s.tester.Run("gfd-normal", func(t *testing.T) {
		s.real.Range(func(key, value interface{}) bool {
			s.Async(t, value.(gfd.GFD))
			return true
		})
	})
	// 检测失效数据-连接已释放，但还拥有gfd
	s.tester.Run("gfd-invalid", func(t *testing.T) {
		s.fail.Range(func(key, value interface{}) bool {
			s.Async(t, value.(gfd.GFD))
			return true
		})
	})
	// 检测假数据-业务端自行生成gfd假数据
	s.tester.Run("gfd-fake", func(t *testing.T) {
		err := s.eng.AsyncWrite(gfd.NewGFD(math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt), []byte{}, nil)
		assert.NoError(s.tester, err)
		err = s.eng.AsyncWrite(gfd.NewGFD(-1, -1, -1, -1), []byte{}, nil)
		assert.NoError(s.tester, err)
		for i := 0; i < 100; i++ {
			err = s.eng.AsyncWrite(gfd.NewGFD(rand.Int(), rand.Int(), rand.Int(), rand.Int()), []byte{}, nil)
			assert.NoError(s.tester, err)
		}
		for elIdx := 0; elIdx < s.elNum; elIdx++ {
			err = s.eng.AsyncWrite(gfd.NewGFD(rand.Int(), elIdx, rand.Int(), rand.Int()), []byte{}, nil)
			assert.NoError(s.tester, err)
		}
	})
	go func() {
		for _, v := range s.client {
			_ = v.Close()
		}
		_ = s.eng.Stop(context.Background())
	}()
	return time.Minute, None
}

func (s *testServerGFD) Async(t *testing.T, gfd gfd.GFD) {
	err := s.eng.AsyncWrite(gfd, []byte("server hello"), nil)
	assert.NoError(t, err)
	err = s.eng.AsyncWritev(gfd, [][]byte{[]byte("server"), []byte("hello")}, nil)
	assert.NoError(t, err)
	err = s.eng.Wake(gfd, nil)
	assert.NoError(t, err)
	err = s.eng.Close(gfd, nil)
	assert.NoError(t, err)
}
