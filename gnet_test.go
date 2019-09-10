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
	t.Run("stdlib", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp-net", ":9997", false, 10, 1, Random)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp-net", ":9998", false, 10, 5, LeastConnections)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp-net", ":9999", false, 10, -1, RoundRobin)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp-net", ":9989", true, 10, 1, Random)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp-net", ":9988", true, 10, 5, LeastConnections)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp-net", ":9987", true, 10, -1, RoundRobin)
			})
		})
	})
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", false, 10, 1, Random)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, 10, 5, LeastConnections)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9993", false, 10, -1, RoundRobin)
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9994", true, 10, 1, Random)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9995", true, 10, 5, LeastConnections)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9996", true, 10, -1, RoundRobin)
			})
		})
	})

}

func testServe(network, addr string, unix bool, nclients, nloops int, balance LoadBalance) {
	var started int32
	var connected int32
	var disconnected int32

	var events Events
	events.LoadBalance = balance
	events.NumLoops = nloops
	events.Serving = func(srv Server) (action Action) {
		return
	}
	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		c.SetContext(c)
		atomic.AddInt32(&connected, 1)
		out = []byte("sweetness\r\n")
		opts.TCPKeepAlive = time.Minute * 5
		if c.LocalAddr() == nil {
			panic("nil local addr")
		}
		if c.RemoteAddr() == nil {
			panic("nil local addr")
		}
		return
	}
	events.Closed = func(c Conn, err error) (action Action) {
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
	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		out = in
		return
	}
	events.Tick = func() (delay time.Duration, action Action) {
		if atomic.LoadInt32(&started) == 0 {
			for i := 0; i < nclients; i++ {
				go startClient(network, addr, nloops)
			}
			atomic.StoreInt32(&started, 1)
		}
		delay = time.Second / 5
		return
	}
	var err error
	if unix {
		socket := strings.Replace(addr, ":", "socket", 1)
		os.RemoveAll(socket)
		defer os.RemoveAll(socket)
		err = Serve(events, network+"://"+addr, "unix://"+socket)
	} else {
		err = Serve(events, network+"://"+addr)
	}
	if err != nil {
		panic(err)
	}
}

func startClient(network, addr string, nloops int) {
	onetwork := network
	network = strings.Replace(network, "-net", "", -1)
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	if err != nil {
		panic(err)
	}
	if string(msg) != "sweetness\r\n" {
		panic("bad header")
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		sz := rand.Int() % (1024 * 1024)
		data := make([]byte, sz)
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
			fmt.Printf("mismatch %s/%d: %d vs %d bytes\n", onetwork, nloops, len(data), len(data2))
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
		testTick("tcp", ":9991", false)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testTick("tcp", ":9992", true)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testTick("unix", "socket1", false)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testTick("unix", "socket2", true)
	}()
	wg.Wait()
}
func testTick(network, addr string, stdlib bool) {
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
	if stdlib {
		must(Serve(events, network+"-net://"+addr))
	} else {
		must(Serve(events, network+"://"+addr))
	}
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
		testShutdown("tcp", ":9991", false)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testShutdown("tcp", ":9992", true)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testShutdown("unix", "socket1", false)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testShutdown("unix", "socket2", true)
	}()
	wg.Wait()
}
func testShutdown(network, addr string, stdlib bool) {
	var events Events
	var count int
	var clients int64
	var N = 10
	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		atomic.AddInt64(&clients, 1)
		return
	}
	events.Closed = func(c Conn, err error) (action Action) {
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
			if int(atomic.LoadInt64(&clients)) == N {
				action = Shutdown
			}
		}
		count++
		delay = time.Second / 20
		return
	}
	if stdlib {
		must(Serve(events, network+"-net://"+addr))
	} else {
		must(Serve(events, network+"://"+addr))
	}
	if clients != 0 {
		panic("did not call close on all clients")
	}
}

func TestDetach(t *testing.T) {
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			testDetach("tcp", ":9991", false)
		})
		t.Run("unix", func(t *testing.T) {
			testDetach("unix", "socket1", false)
		})
	})
	t.Run("stdlib", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			testDetach("tcp", ":9992", true)
		})
		t.Run("unix", func(t *testing.T) {
			testDetach("unix", "socket2", true)
		})
	})
}

func testDetach(network, addr string, stdlib bool) {
	// we will write a bunch of data with the text "--detached--" in the
	// middle followed by a bunch of data.
	rand.Seed(time.Now().UnixNano())
	rdat := make([]byte, 10*1024)
	if _, err := rand.Read(rdat); err != nil {
		panic("random error: " + err.Error())
	}
	expected := []byte(string(rdat) + "--detached--" + string(rdat))
	var cin []byte
	var events Events
	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		cin = append(cin, in...)
		if len(cin) >= len(expected) {
			if string(cin) != string(expected) {
				panic("mismatch client -> server")
			}
			return cin, Detach
		}
		return
	}

	var done int64
	events.Detached = func(c Conn, conn io.ReadWriteCloser) (action Action) {
		go func() {
			p := make([]byte, len(expected))
			defer conn.Close()
			_, err := io.ReadFull(conn, p)
			must(err)
			conn.Write(expected)
		}()
		return
	}

	events.Serving = func(srv Server) (action Action) {
		go func() {
			p := make([]byte, len(expected))
			_ = expected
			conn, err := net.Dial(network, addr)
			must(err)
			defer conn.Close()
			conn.Write(expected)
			_, err = io.ReadFull(conn, p)
			must(err)
			conn.Write(expected)
			_, err = io.ReadFull(conn, p)
			must(err)
			atomic.StoreInt64(&done, 1)
		}()
		return
	}
	events.Tick = func() (delay time.Duration, action Action) {
		delay = time.Second / 5
		if atomic.LoadInt64(&done) == 1 {
			action = Shutdown
		}
		return
	}
	if stdlib {
		must(Serve(events, network+"-net://"+addr))
	} else {
		must(Serve(events, network+"://"+addr))
	}
}

func TestBadAddresses(t *testing.T) {
	var events Events
	events.Serving = func(srv Server) (action Action) {
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

func TestInputStream(t *testing.T) {
	var s InputStream
	in := []byte("HELLO")
	data := s.Begin(in)
	if string(data) != string(in) {
		t.Fatalf("expected '%v', got '%v'", in, data)
	}
	s.End(in[3:])
	data = s.Begin([]byte("WLY"))
	if string(data) != "LOWLY" {
		t.Fatalf("expected '%v', got '%v'", "LOWLY", data)
	}
	s.End(nil)
	data = s.Begin([]byte("PLAYER"))
	if string(data) != "PLAYER" {
		t.Fatalf("expected '%v', got '%v'", "PLAYER", data)
	}
}

func TestReuseInputBuffer(t *testing.T) {
	reuses := []bool{true, false}
	for _, reuse := range reuses {
		var events Events
		events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
			opts.ReuseInputBuffer = reuse
			return
		}
		var prev []byte
		events.Data = func(c Conn, in []byte) (out []byte, action Action) {
			if prev == nil {
				prev = in
			} else {
				reused := string(in) == string(prev)
				if reused != reuse {
					t.Fatalf("expected %v, got %v", reuse, reused)
				}
				action = Shutdown
			}
			return
		}
		events.Serving = func(_ Server) (action Action) {
			go func() {
				c, err := net.Dial("tcp", ":9991")
				must(err)
				defer c.Close()
				c.Write([]byte("packet1"))
				time.Sleep(time.Second / 5)
				c.Write([]byte("packet2"))
			}()
			return
		}
		must(Serve(events, "tcp://:9991"))
	}

}

func TestReuseport(t *testing.T) {
	var events Events
	events.Serving = func(s Server) (action Action) {
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
