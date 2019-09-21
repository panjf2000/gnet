// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gnet

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	// start a server
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the server will be random sizes. 0KB - 1MB.
	// the server will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", false, false, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, false, true, 10)
			})
		})
		t.Run("tcp-unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9993", true, false, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9994", true, false, true, 10)
			})
		})

		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9995", false, false, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9996", false, false, true, 10)
			})
		})
		t.Run("udp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9997", true, false, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9998", true, false, true, 10)
			})
		})
	})

	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", false, true, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, true, true, 10)
			})
		})
		t.Run("tcp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9993", true, true, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9994", true, true, true, 10)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9995", false, true, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9996", false, true, true, 10)
			})
		})
		t.Run("udp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9997", true, true, false, 10)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9998", true, true, true, 10)
			})
		})
	})
}

func testServe(network, addr string, unix, reuseport bool, multicore bool, nclients int) {
	var once sync.Once
	var started int32
	var connected int32
	var clientActive int32
	var disconnected int32

	var events Events
	events.Multicore = multicore
	//events.OnInitComplete = func(srv Server) (action Action) {
	//	return
	//}
	events.OnOpened = func(c Conn) (opts Options, action Action) {
		c.SetContext(c)
		atomic.AddInt32(&connected, 1)
		out := []byte("sweetness\r\n")
		c.Write(out)
		opts.TCPKeepAlive = time.Minute * 5
		if c.LocalAddr() == nil {
			panic("nil local addr")
		}
		if c.RemoteAddr() == nil {
			panic("nil local addr")
		}
		return
	}
	events.OnClosed = func(c Conn, err error) (action Action) {
		if c.Context() != c {
			panic("invalid context")
		}

		atomic.AddInt32(&disconnected, 1)
		if atomic.LoadInt32(&connected) == atomic.LoadInt32(&disconnected) &&
			atomic.LoadInt32(&disconnected) == int32(nclients) {
			action = Shutdown
		}
		return
	}
	events.React = func(c Conn) (action Action) {
		top, tail := c.Read()
		c.Write(append(top, tail...))
		c.ResetBuffer()
		once.Do(func() {
			if !reuseport {
				c.Wake()
			}
		})
		return
	}
	events.Tick = func() (delay time.Duration, action Action) {
		if atomic.LoadInt32(&started) == 0 {
			for i := 0; i < nclients; i++ {
				atomic.AddInt32(&clientActive, 1)
				go func() {
					startClient(network, addr, multicore)
					atomic.AddInt32(&clientActive, -1)
				}()
			}
			atomic.StoreInt32(&started, 1)
		}
		if network == "udp" && atomic.LoadInt32(&clientActive) == 0 {
			action = Shutdown
			return
		}
		delay = time.Second / 5
		return
	}
	var err error
	if unix {
		socket := strings.Replace(addr, ":", "socket", 1)
		os.RemoveAll(socket)
		defer os.RemoveAll(socket)
		if reuseport {
			err = Serve(events, network+"://"+addr+"?reuseport=t", "unix://"+socket)
		} else {
			err = Serve(events, network+"://"+addr, "unix://"+socket)
		}
	} else {
		if reuseport {
			err = Serve(events, network+"://"+addr+"?reuseport=t")
		} else {
			err = Serve(events, network+"://"+addr)
		}
	}
	if err != nil {
		panic(err)
	}
}

func startClient(network, addr string, multicore bool) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	rd := bufio.NewReader(c)
	if network != "udp" {
		msg, err := rd.ReadBytes('\n')
		if err != nil {
			panic(err)
		}
		if string(msg) != "sweetness\r\n" {
			panic("bad header")
		}
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		sz := rand.Int() % (1024 * 1024)
		data := make([]byte, sz)
		if network == "udp" {
			n := 64
			if sz < 64 {
				n = sz
			}
			data = data[:n]
		}
		if _, err := rand.Read(data); err != nil {
			panic(err)
		}
		if _, err := c.Write(data); err != nil {
			panic(err)
		}
		data2 := make([]byte, len(data))
		if _, err := io.ReadFull(rd, data2); err != nil {
			panic(err)
		}
		if string(data) != string(data2) {
			fmt.Printf("mismatch %s/multi-core:%t: %d vs %d bytes\n", network, multicore, len(data), len(data2))
			//panic("mismatch")
		}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func TestTick(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		testTick("tcp", ":9991")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testTick("unix", "socket1")
	}()
	wg.Wait()
}
func testTick(network, addr string) {
	var events Events
	var count int
	start := time.Now()
	events.Tick = func() (delay time.Duration, action Action) {
		if count == 25 {
			action = Shutdown
			return
		}
		count++
		delay = time.Millisecond * 10
		return
	}
	must(Serve(events, network+"://"+addr))
	dur := time.Since(start)
	if dur < 250&time.Millisecond || dur > time.Second {
		panic("bad ticker timing")
	}
}

func TestShutdown(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		testShutdown("tcp", ":9991")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testShutdown("unix", "socket1")
	}()
	wg.Wait()
}
func testShutdown(network, addr string) {
	var events Events
	var count int
	var clients int64
	var N = 10
	events.OnOpened = func(c Conn) (opts Options, action Action) {
		atomic.AddInt64(&clients, 1)
		return
	}
	events.OnClosed = func(c Conn, err error) (action Action) {
		atomic.AddInt64(&clients, -1)
		return
	}
	events.Tick = func() (delay time.Duration, action Action) {
		if count == 0 {
			// start clients
			for i := 0; i < N; i++ {
				go func() {
					conn, err := net.Dial(network, addr)
					must(err)
					defer conn.Close()
					_, err = conn.Read([]byte{0})
					if err == nil {
						panic("expected error")
					}
				}()
			}
		} else {
			fmt.Printf("ticker clients: %d\n", atomic.LoadInt64(&clients))
			if int(atomic.LoadInt64(&clients)) == N {
				fmt.Printf("ticker shutdown...\n")
				action = Shutdown
			}
		}
		count++
		delay = time.Second / 20
		return
	}
	must(Serve(events, network+"://"+addr))
	if clients != 0 {
		panic("did not call close on all clients")
	}
}

func TestBadAddresses(t *testing.T) {
	var events Events
	events.OnInitComplete = func(srv Server) (action Action) {
		return Shutdown
	}
	if err := Serve(events, "tulip://howdy"); err == nil {
		t.Fatalf("expected error")
	}
	if err := Serve(events, "howdy"); err == nil {
		t.Fatalf("expected error")
	}
	if err := Serve(events, "tcp://"); err != nil {
		t.Fatalf("expected nil, got '%v'", err)
	}
}

func TestReuseport(t *testing.T) {
	var events Events
	events.OnInitComplete = func(s Server) (action Action) {
		return Shutdown
	}
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		var t = "1"
		if i%2 == 0 {
			t = "true"
		}
		go func(t string) {
			defer wg.Done()
			must(Serve(events, "tcp://:9991?reuseport="+t))
		}(t)
	}
	wg.Wait()
}
