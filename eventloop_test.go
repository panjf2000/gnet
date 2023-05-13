// Copyright (c) 2023
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gnet

import (
	"context"
	"net"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/assert"
)

func (lb *roundRobinLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
	registerInitConn(el)
}

func (lb *leastConnectionsLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
	registerInitConn(el)
}

func (lb *sourceAddrHashLoadBalancer) register(el *eventloop) {
	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
	registerInitConn(el)
}

func registerInitConn(el *eventloop) {
	for i := 0; i < int(atomic.LoadInt32(&nowEventLoopInitConn)); i++ {
		c := newTCPConn(i, el, &unix.SockaddrInet4{}, &net.TCPAddr{}, &net.TCPAddr{})
		el.connections.storeConn(c, el.idx)
	}
}

var nowEventLoopInitConn int32

// TestServeGC generate fake data asynchronously, if you need to test, manually open the comment.
func TestServeGC(t *testing.T) {
	t.Run("gc-loop", func(t *testing.T) {
		t.Run("1-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 1, 10000)
		})
		// t.Run("1-loop-100000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 1, 100000)
		// })
		// t.Run("1-loop-1000000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 1, 1000000)
		// })
		t.Run("2-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 2, 10000)
		})
		// t.Run("2-loop-100000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 2, 100000)
		// })
		// t.Run("2-loop-1000000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 2, 1000000)
		// })
		t.Run("4-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 4, 10000)
		})
		// t.Run("4-loop-100000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 4, 100000)
		// })
		// t.Run("4-loop-1000000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 4, 1000000)
		// })
		t.Run("16-loop-10000", func(t *testing.T) {
			testServeGC(t, "tcp", ":9000", true, true, 16, 10000)
		})
		// t.Run("16-loop-100000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 16, 100000)
		// })
		// t.Run("16-loop-1000000", func(t *testing.T) {
		// 	testServeGC(t, "tcp", ":9000", true, true, 16, 1000000)
		// })
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
	go s.GC()

	return
}

func (s *testServerGC) GC() {
	defer func() {
		_ = s.eng.Stop(context.Background())
		runtime.GC()
	}()
	var gcAllTime, gcAllCount time.Duration
	gcStart := time.Now()
	for range time.Tick(time.Second * 2) {
		gcAllCount++
		now := time.Now()
		runtime.GC()
		gcTime := time.Since(now)
		gcAllTime += gcTime
		s.tester.Log(s.tester.Name(), s.network, " server gc:", gcTime, ", average gc time: ", gcAllTime/gcAllCount)
		if time.Since(gcStart) >= time.Second*10 {
			break
		}
	}
}
