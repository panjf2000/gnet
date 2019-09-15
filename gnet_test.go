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

	"github.com/panjf2000/gnet/ringbuffer"
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
				testServe("tcp", ":9991", false, false, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, false, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9993", false, false, 10, -1)
			})
		})
		t.Run("tcp-unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9994", true, false, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9995", true, false, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9996", true, false, 10, -1)
			})
		})

		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9997", false, false, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("udp", ":9998", false, false, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9999", false, false, 10, -1)
			})
		})
		t.Run("udp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9981", true, false, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("udp", ":9982", true, false, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9983", true, false, 10, -1)
			})
		})
	})

	t.Run("poll-reuseport", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9991", false, true, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9992", false, true, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9993", false, true, 10, -1)
			})
		})
		t.Run("tcp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("tcp", ":9994", true, true, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("tcp", ":9995", true, true, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("tcp", ":9996", true, true, 10, -1)
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9997", false, true, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("udp", ":9998", false, true, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9999", false, true, 10, -1)
			})
		})
		t.Run("udp-unix-reuseport", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				testServe("udp", ":9981", true, true, 10, 1)
			})
			t.Run("5-loop", func(t *testing.T) {
				testServe("udp", ":9982", true, true, 10, 5)
			})
			t.Run("N-loop", func(t *testing.T) {
				testServe("udp", ":9983", true, true, 10, -1)
			})
		})
	})
}

func testServe(network, addr string, unix, reuseport bool, nclients, nloops int) {
	var started int32
	var connected int32
	var clientActive int32
	var disconnected int32

	var events Events
	events.NumLoops = nloops
	//events.OnInitComplete = func(srv Server) (action Action) {
	//	return
	//}
	events.OnOpened = func(c Conn) (out []byte, opts Options, action Action) {
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
	events.OnClosed = func(c Conn, err error) (action Action) {
		if c.Context() != c {
			panic("invalid context")
		}

		atomic.AddInt32(&disconnected, 1)
		if atomic.LoadInt32(&connected) == atomic.LoadInt32(&disconnected) &&
			atomic.LoadInt32(&disconnected) == int32(nclients) {
			action = Shutdown
		}
		//fmt.Printf("connection closing, action: %v\n", action)
		return
	}
	events.React = func(c Conn, inBuf *ringbuffer.RingBuffer) (out []byte, action Action) {
		n := inBuf.Length()
		out = inBuf.Bytes()
		inBuf.Advance(n)
		return
	}
	events.Tick = func() (delay time.Duration, action Action) {
		if atomic.LoadInt32(&started) == 0 {
			for i := 0; i < nclients; i++ {
				atomic.AddInt32(&clientActive, 1)
				//fmt.Println("start client...")
				go func() {
					startClient(network, addr, nloops)
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

func startClient(network, addr string, nloops int) {
	onetwork := network
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
	events.OnOpened = func(c Conn) (out []byte, opts Options, action Action) {
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

func TestDetach(t *testing.T) {
	t.Run("poll", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			testDetach("tcp", ":9991")
		})
		t.Run("unix", func(t *testing.T) {
			testDetach("unix", "socket1")
		})
	})
}

func testDetach(network, addr string) {
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
	events.React = func(c Conn, inBuf *ringbuffer.RingBuffer) (out []byte, action Action) {
		n := inBuf.Length()
		cin = append(cin, inBuf.Bytes()...)
		inBuf.Advance(n)
		if len(cin) >= len(expected) {
			if string(cin) != string(expected) {
				panic("mismatch client -> server")
			}
			return cin, Detach
		}
		return
	}

	var done int64
	events.OnDetached = func(c Conn, conn io.ReadWriteCloser) (action Action) {
		go func() {
			p := make([]byte, len(expected))
			defer conn.Close()
			_, err := io.ReadFull(conn, p)
			must(err)
			_, _ = conn.Write(expected)
		}()
		return
	}

	events.OnInitComplete = func(srv Server) (action Action) {
		go func() {
			p := make([]byte, len(expected))
			_ = expected
			conn, err := net.Dial(network, addr)
			must(err)
			defer conn.Close()
			_, _ = conn.Write(expected)
			_, err = io.ReadFull(conn, p)
			must(err)
			_, _ = conn.Write(expected)
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
	must(Serve(events, network+"://"+addr))
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
