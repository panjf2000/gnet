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

// TestServeGC generate fake data asynchronously, there will be a race condition, if you need to test, manually open the comment.
// func TestServeGC(t *testing.T) {
//	t.Run("gc-loop", func(t *testing.T) {
//		t.Run("1-loop-10000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 1, 10000)
//		})
//		t.Run("1-loop-100000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 1, 100000)
//		})
//		t.Run("1-loop-1000000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 1, 1000000)
//		})
//		t.Run("2-loop-10000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 2, 10000)
//		})
//		t.Run("2-loop-100000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 2, 100000)
//		})
//		t.Run("2-loop-1000000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 2, 1000000)
//		})
//		t.Run("4-loop-10000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 4, 10000)
//		})
//		t.Run("4-loop-100000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 4, 100000)
//		})
//		t.Run("4-loop-1000000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 4, 1000000)
//		})
//		t.Run("16-loop-10000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 16, 10000)
//		})
//		t.Run("16-loop-100000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 16, 100000)
//		})
//		t.Run("16-loop-1000000", func(t *testing.T) {
//			testServeGC(t, "tcp", ":9000", true, true, 16, 1000000)
//		})
//	})
// }
//
// func testServeGC(t *testing.T, network, addr string, multicore, async bool, elNum, initConnCount int) {
//	ts := &testServerGC{
//		tester:        t,
//		network:       network,
//		addr:          addr,
//		multicore:     multicore,
//		async:         async,
//		workerPool:    goPool.Default(),
//		elNum:         elNum,
//		initConnCount: initConnCount,
//	}
//
//	err := Run(ts,
//		network+"://"+addr,
//		WithLockOSThread(async),
//		WithMulticore(multicore),
//		WithNumEventLoop(elNum),
//		WithTCPKeepAlive(time.Minute*1),
//		WithTCPNoDelay(TCPDelay))
//	assert.NoError(t, err)
// }
//
// type testServerGC struct {
//	*BuiltinEventEngine
//	tester        *testing.T
//	eng           Engine
//	network       string
//	addr          string
//	multicore     bool
//	async         bool
//	workerPool    *goPool.Pool
//	elNum         int
//	initConnCount int
// }
//
// func (s *testServerGC) OnBoot(eng Engine) (action Action) {
//	s.eng = eng
//	addr := eng.eng.ln.addr
//	go func() {
//		defer func() {
//			_ = eng.Stop(context.Background())
//			runtime.GC()
//		}()
//		for eng.eng.lb.len() != s.elNum {
//			time.Sleep(time.Second)
//		}
//		eng.eng.lb.iterate(func(idx int, e *eventloop) bool {
//			for i := 0; i < s.initConnCount; i++ {
//				c := newTCPConn(i, e, nil, addr, addr)
//				e.storeConn(c)
//			}
//			return true
//		})
//
//		var gcAllTime, gcAllCount time.Duration
//		gcStart := time.Now()
//		for range time.Tick(time.Second * 2) {
//			gcAllCount++
//			now := time.Now()
//			runtime.GC()
//			gcTime := time.Since(now)
//			gcAllTime += gcTime
//			s.tester.Log(s.tester.Name(), s.network, " server gc:", gcTime, ", average gc time: ", gcAllTime/gcAllCount, ",conn count:", eng.CountConnections())
//			if time.Since(gcStart) >= time.Second*10 {
//				break
//			}
//		}
//	}()
//
//	return
// }
//
// func (s *testServerGC) OnOpen(_ Conn) (out []byte, action Action) {
//	return
// }
//
// func (s *testServerGC) OnClose(_ Conn, _ error) (action Action) {
//	return
// }
//
// func (s *testServerGC) OnTraffic(_ Conn) (action Action) {
//	return
// }
//
// func (s *testServerGC) OnTick() (delay time.Duration, action Action) {
//	return
// }
