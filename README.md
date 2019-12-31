<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square&logo=appveyor"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square&logo=appveyor"></a>
<a title="Supported Platforms" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-549688?style=flat-square&logo=appveyor"></a>
<a title="Require Go Version" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/go-%3E%3D1.9-30dff3?style=flat-square&logo=appveyor"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?color=161823&style=flat-square&logo=appveyor"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<a title="Doc for gnet" target="_blank" href="https://gowalker.org/github.com/panjf2000/gnet?lang=en-US"><img src="https://img.shields.io/badge/api-reference-8d4bbb.svg?style=flat-square&logo=appveyor"></a>
<a title="gnet on Sourcegraph" target="_blank" href="https://sourcegraph.com/github.com/panjf2000/gnet?badge"><img src="https://sourcegraph.com/github.com/panjf2000/gnet/-/badge.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go#networking"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
</p>

English | [🇨🇳中文](README_ZH.md)

# 📖 Introduction

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP/Unix-Socket protocols, so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

# 🚀 Features

- [x] [High-performance](#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in load balancing algorithm: Round-Robin
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Built-in memory pool with bytes powered by the library [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [x] Concise APIs
- [x] Efficient memory usage: Ring-Buffer
- [x] Supporting multiple protocols: TCP, UDP, and Unix Sockets
- [x] Supporting two event-driven mechanisms: epoll on Linux and kqueue on FreeBSD
- [x] Supporting asynchronous write operation
- [x] Flexible ticker event
- [x] SO_REUSEPORT socket option
- [x] Built-in multiple codecs to encode/decode network frames into/from TCP stream: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://github.com/netty/netty/tree/netty-4.1.43.Final/codec/src/main/java/io/netty/handler/codec), also supporting customized codecs
- [x] Supporting Windows platform with ~~event-driven mechanism of IOCP~~ Go stdlib: net
- [ ] Additional load-balancing algorithms: Random, Least-Connections, Consistent-hashing and so on
- [ ] TLS support
- [ ] Implementation of `gnet` Client

# 💡 Key Designs

## Networking Model of Multiple Threads/Goroutines
### Multiple Reactors

`gnet` redesigns and implements a new built-in networking model of multiple threads/goroutines: 『multiple reactors』 which is also the default networking model of multiple threads in `netty`, Here's the schematic diagram:

<p align="center">
<img alt="multi_reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img alt="reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors-sequence-diagram.png">
</p>

### Multiple Reactors + Goroutine-Pool

You may ask me a question: what if my business logic in `EventHandler.React`  contains some blocking code which leads to blocking in event-loop of `gnet`, what is the solution for this kind of situation？

As you know, there is a most important tenet when writing code under `gnet`: you should never block the event-loop goroutine in the `EventHandler.React`, which is also the most important tenet in `netty`, otherwise, it will result in a low throughput in your `gnet` server.

And the solution to that could be found in the subsequent networking model of multiple threads/goroutines in `gnet`: 『multiple reactors with thread/goroutine pool』which pulls you out from the blocking mire, it will construct a worker-pool with fixed capacity and put those blocking jobs in `EventHandler.React` into the worker-pool to make the event-loop goroutines non-blocking.

The networking model:『multiple reactors with thread/goroutine pool』dissolves the blocking jobs by introducing a goroutine pool, as shown below:

<p align="center">
<img alt="multi_reactor_thread_pool" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img alt="multi-reactors" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool-sequence-diagram.png">
</p>

`gnet` implements the networking model:『multiple reactors with thread/goroutine pool』by the aid of a high-performance goroutine pool called [ants](https://github.com/panjf2000/ants) that allows you to manage and recycle a massive number of goroutines in your concurrent programs, the full features and usages in `ants` are documented [here](https://gowalker.org/github.com/panjf2000/ants?lang=en-US).

`gnet` integrates `ants` and provides the `pool.goroutine.Default()` method that you can invoke to instantiate a `ants` pool where you are able to put your blocking code logic and invoke the function `gnet.Conn.AsyncWrite([]byte)` to send out data asynchronously after you finish the blocking process and get the output data, which makes the goroutine of event-loop non-blocking.

The details about integrating `gnet`  with `ants` are shown [here](#echo-server-with-blocking-logic).

## Auto-scaling Ring Buffer

There are two ring-buffers inside `gnet`: inbound buffer and outbound buffer to buffer and manage inbound/outbound network data.

The purpose of implementing inbound and outbound ring-buffers in `gnet` is to transfer the logic of buffering and managing network data based on application protocol upon TCP stream from business server to framework and unify the network data buffer, which minimizes the complexity of business code so that developers are able to concentrate on business logic instead of the underlying implementation.

<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif">
</p>


# 🎉 Getting Started

## Prerequisites

`gnet` requires Go 1.9 or later.

## Installation

```powershell
go get -u github.com/panjf2000/gnet
```

`gnet` is available as a Go module, with [Go 1.11 Modules](https://github.com/golang/go/wiki/Modules) support (Go 1.11+), just simply `import "github.com/panjf2000/gnet"` in your source code and `go [build|run|test]` will download the necessary dependencies automatically.

## Usage Examples

**The detailed documentation is located in here: [docs of gnet](https://gowalker.org/github.com/panjf2000/gnet?lang=en-US), but let's pass through the brief instructions first.**

It is easy to create a network server with `gnet`. All you have to do is just to make your implementation of `gnet.EventHandler` interface and register your event-handler functions to it, then pass it to the `gnet.Serve` function along with the binding address(es). Each connection is represented as a `gnet.Conn` interface that is passed to various events to differentiate the clients. At any point you can close a connection or shutdown the server by return a `Close` or `Shutdown` action from an event function.

The simplest example to get you started playing with `gnet` would be the echo server. So here you are, a simplest echo server upon `gnet` that is listening on port 9000:

### Echo server without blocking logic

<details>
	<summary> Old version(<=v1.0.0-rc.4)  </summary>

```go
package main

import (
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) React(c gnet.Conn) (out []byte, action gnet.Action) {
	out = c.Read()
	c.ResetBuffer()
	return
}

func main() {
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```
</details>

```go
package main

import (
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	out = frame
	return
}

func main() {
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```

As you can see, this example of echo server only sets up the `EventHandler.React` function where you commonly write your main business code and it will be invoked once the server receives input data from a client. The output data will be then sent back to that client by assigning the `out` variable and return it after your business code finish processing data(in this case, it just echo the data back).

### Echo server with blocking logic

<details>
	<summary> Old version(<=v1.0.0-rc.4)  </summary>

```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type echoServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (es *echoServer) React(c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, c.Read()...)
	c.ResetBuffer()

	// Use ants pool to unblock the event-loop.
	_ = es.pool.Submit(func() {
		time.Sleep(1 * time.Second)
		c.AsyncWrite(data)
	})

	return
}

func main() {
	p := goroutine.Default()
	defer p.Release()
	
	echo := &echoServer{pool: p}
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```
</details>

```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type echoServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)

	// Use ants pool to unblock the event-loop.
	_ = es.pool.Submit(func() {
		time.Sleep(1 * time.Second)
		c.AsyncWrite(data)
	})

	return
}

func main() {
	p := goroutine.Default()
	defer p.Release()

	echo := &echoServer{pool: p}
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```

Like I said in the 『Multiple Reactors + Goroutine-Pool』section, if there are blocking code in your business logic, then you ought to turn them into non-blocking code in any way, for instance you can wrap them into a goroutine, but it will result in a massive amount of goroutines if massive traffic is passing through your server so I would suggest you utilize a goroutine pool like [ants](https://github.com/panjf2000/ants) to manage those goroutines and reduce the cost of system resources.

**All gnet examples:**

<details>
	<summary> TCP Echo Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Echo server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}
func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	out = frame
	return

	/*
		// Echo asynchronously.
		data := append([]byte{}, frame...)
		go func() {
			time.Sleep(time.Second)
			c.AsyncWrite(data)
		}()
		return
	*/
}

func main() {
	var port int
	var multicore bool

	// Example command: go run echo.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.Parse()
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore)))
}
```
</details>

<details>
	<summary> UDP Echo Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("UDP Echo server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}
func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	out = frame
	return

	/*
		// Echo asynchronously.
		data := append([]byte{}, frame...)
		go func() {
			time.Sleep(time.Second)
			c.SendTo(data)
		}()
		return
	*/
}

func main() {
	var port int
	var multicore, reuseport bool

	// Example command: go run echo.go --port 9000 --multicore=true --reuseport=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.BoolVar(&reuseport, "reuseport", false, "--reuseport true")
	flag.Parse()
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("udp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport)))
}
```
</details>

<details>
	<summary> HTTP Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/panjf2000/gnet"
)

var res string
var resBytes []byte

type request struct {
	proto, method string
	path, query   string
	head, body    string
	remoteAddr    string
}

type httpServer struct {
	*gnet.EventServer
}

var errMsg = "Internal Server Error"
var errMsgBytes = []byte(errMsg)

type httpCodec struct {
	req request
}

func (hc *httpCodec) Encode(c gnet.Conn, buf []byte) (out []byte, err error) {
	if c.Context() == nil {
		return appendHandle(out, res), nil
	}
	return appendResp(out, "500 Error", "", errMsg+"\n"), nil
}

func (hc *httpCodec) Decode(c gnet.Conn) ([]byte, error) {
	buf := c.Read()
	// process the pipeline
	leftover, err := parseReq(buf, &hc.req)
	// bad thing happened
	if err != nil {
		c.SetContext(err)
		return nil, err
	} else if len(leftover) == len(buf) {
		// request not ready, yet
		return nil, nil
	}
	c.ResetBuffer()
	return buf, nil
}

func (hs *httpServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("HTTP server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}

func (hs *httpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// process the pipeline
	if c.Context() != nil {
		// bad thing happened
		out = errMsgBytes
		action = gnet.Close
		return
	}
	// handle the request
	out = resBytes
	return
}

func main() {
	var port int
	var multicore bool

	// Example command: go run http.go --port 8080 --multicore=true
	flag.IntVar(&port, "port", 8080, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	res = "Hello World!\r\n"
	resBytes = []byte(res)

	http := new(httpServer)
	hc := new(httpCodec)

	// Start serving!
	log.Fatal(gnet.Serve(http, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithCodec(hc)))
}

// appendHandle handles the incoming request and appends the response to
// the provided bytes, which is then returned to the caller.
func appendHandle(b []byte, res string) []byte {
	return appendResp(b, "200 OK", "", res)
}

// appendResp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func appendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: gnet\r\n"...)
	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	b = append(b, head...)
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// parseReq is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parseReq(data []byte, req *request) (leftover []byte, err error) {
	sdata := string(data)
	var i, s int
	var head string
	var clen int
	var q = -1
	// method, path, proto line
	for ; i < len(sdata); i++ {
		if sdata[i] == ' ' {
			req.method = sdata[s:i]
			for i, s = i+1, i+1; i < len(sdata); i++ {
				if sdata[i] == '?' && q == -1 {
					q = i - s
				} else if sdata[i] == ' ' {
					if q != -1 {
						req.path = sdata[s:q]
						req.query = req.path[q+1 : i]
					} else {
						req.path = sdata[s:i]
					}
					for i, s = i+1, i+1; i < len(sdata); i++ {
						if sdata[i] == '\n' && sdata[i-1] == '\r' {
							req.proto = sdata[s:i]
							i, s = i+1, i+1
							break
						}
					}
					break
				}
			}
			break
		}
	}
	if req.proto == "" {
		return data, fmt.Errorf("malformed request")
	}
	head = sdata[:s]
	for ; i < len(sdata); i++ {
		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
			line := sdata[s : i-1]
			s = i + 1
			if line == "" {
				req.head = sdata[len(head)+2 : i+1]
				i++
				if clen > 0 {
					if len(sdata[i:]) < clen {
						break
					}
					req.body = sdata[i : i+clen]
					i += clen
				}
				return data[i:], nil
			}
			if strings.HasPrefix(line, "Content-Length:") {
				n, err := strconv.ParseInt(strings.TrimSpace(line[len("Content-Length:"):]), 10, 64)
				if err == nil {
					clen = int(n)
				}
			}
		}
	}
	// not enough data
	return data, nil
}
```
</details>

<details>
	<summary> Push Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/panjf2000/gnet"
)

type pushServer struct {
	*gnet.EventServer
	tick             time.Duration
	connectedSockets sync.Map
}

func (ps *pushServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Push server is listening on %s (multi-cores: %t, loops: %d), "+
		"pushing data every %s ...\n", srv.Addr.String(), srv.Multicore, srv.NumLoops, ps.tick.String())
	return
}
func (ps *pushServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Printf("Socket with addr: %s has been opened...\n", c.RemoteAddr().String())
	ps.connectedSockets.Store(c.RemoteAddr().String(), c)
	return
}
func (ps *pushServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Printf("Socket with addr: %s is closing...\n", c.RemoteAddr().String())
	ps.connectedSockets.Delete(c.RemoteAddr().String())
	return
}
func (ps *pushServer) Tick() (delay time.Duration, action gnet.Action) {
	log.Println("It's time to push data to clients!!!")
	ps.connectedSockets.Range(func(key, value interface{}) bool {
		addr := key.(string)
		c := value.(gnet.Conn)
		c.AsyncWrite([]byte(fmt.Sprintf("heart beating to %s\n", addr)))
		return true
	})
	delay = ps.tick
	return
}
func (ps *pushServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	out = frame
	return
}

func main() {
	var port int
	var multicore bool
	var interval time.Duration
	var ticker bool

	// Example command: go run push.go --port 9000 --tick 1s --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.DurationVar(&interval, "tick", 0, "pushing tick")
	flag.Parse()
	if interval > 0 {
		ticker = true
	}
	push := &pushServer{tick: interval}
	log.Fatal(gnet.Serve(push, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithTicker(ticker)))
}
```
</details>

<details>
	<summary> Codec Client/Server </summary>

**Client:**

```go
// Reference https://github.com/smallnest/goframe/blob/master/_examples/goclient/client.go

package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/smallnest/goframe"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	encoderConfig := goframe.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}

	decoderConfig := goframe.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	fc := goframe.NewLengthFieldBasedFrameConn(encoderConfig, decoderConfig, conn)
	err = fc.WriteFrame([]byte("hello"))
	if err != nil {
		panic(err)
	}
	err = fc.WriteFrame([]byte("world"))
	if err != nil {
		panic(err)
	}

	buf, err := fc.ReadFrame()
	if err != nil {
		panic(err)
	}
	fmt.Println("received: ", string(buf))
	buf, err = fc.ReadFrame()
	if err != nil {
		panic(err)
	}
	fmt.Println("received: ", string(buf))
}
```

**Server:**

```go
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type codecServer struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func (cs *codecServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Test codec server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}

func (cs *codecServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if cs.async {
		data := append([]byte{}, frame...)
		_ = cs.workerPool.Submit(func() {
			c.AsyncWrite(data)
		})
		return
	}
	out = frame
	return
}

func testCodecServe(addr string, multicore, async bool, codec gnet.ICodec) {
	var err error
	if codec == nil {
		encoderConfig := gnet.EncoderConfig{
			ByteOrder:                       binary.BigEndian,
			LengthFieldLength:               4,
			LengthAdjustment:                0,
			LengthIncludesLengthFieldLength: false,
		}
		decoderConfig := gnet.DecoderConfig{
			ByteOrder:           binary.BigEndian,
			LengthFieldOffset:   0,
			LengthFieldLength:   4,
			LengthAdjustment:    0,
			InitialBytesToStrip: 4,
		}
		codec = gnet.NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	}
	cs := &codecServer{addr: addr, multicore: multicore, async: async, codec: codec, workerPool: goroutine.Default()}
	err = gnet.Serve(cs, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(codec))
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	var multicore bool

	// Example command: go run server.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()
	addr := fmt.Sprintf("tcp://:%d", port)
	testCodecServe(addr, multicore, false, nil)
}
```
</details>

**For more details, check out here: [examples of gnet](https://github.com/panjf2000/gnet/tree/master/examples).**

## I/O Events

Current supported I/O events in `gnet`:

- `EventHandler.OnInitComplete` fires when the server has been initialized and ready to accept new connections.
- `EventHandler.OnOpened` fires once a connection has been opened.
- `EventHandler.OnClosed` fires after a connection has been closed.
- `EventHandler.React` fires when the server receives inbound data from a socket/connection. (usually it is where you write the code of business logic)
- `EventHandler.Tick` fires right after the server starts and then fires every specified interval.
- `EventHandler.PreWrite` fires just before any data has been written to client.


## Ticker

The `EventHandler.Tick` event fires ticks at a specified interval. 
The first tick fires right after the gnet server starts up and if you intend to set up a ticker event, don't forget to pass an option: `gnet.WithTicker(true)` to `gnet.Serve`.

```go
events.Tick = func() (delay time.Duration, action Action){
	log.Printf("tick")
	delay = time.Second
	return
}
```

## UDP

The `gnet.Serve` function can bind to UDP addresses. 

- All incoming and outgoing packets will not be buffered but sent individually.
- The `EventHandler.OnOpened` and `EventHandler.OnClosed` events are not available for UDP sockets, only the `React` event.
- The UDP equivalents of  `Read()/ReadFrame()` and `AsyncWrite([]byte)` in TCP are `ReadFromUDP()` and `SendTo([]byte)`.

## Multi-threads

The `gnet.WithMulticore(true)` indicates whether the server will be effectively created with multi-cores, if so, then you must take care of synchronizing memory between all event callbacks, otherwise, it will run the server with a single thread. The number of threads in the server will be automatically assigned to the value of `runtime.NumCPU()`.

## Load Balancing

The current built-in load balancing algorithm in `gnet` is Round-Robin.

## SO_REUSEPORT

`gnet` server is able to utilize the [SO_REUSEPORT](https://lwn.net/Articles/542629/) option which allows multiple sockets on the same host to bind to the same port and the OS kernel takes care of the load balancing for you, it wakes one socket per `accpet` event coming to resolved the `thundering herd`.

By default, `gnet` is not going to be haunted by the `thundering herd` under its networking model:『multiple reactors』which gets only **one** main reactor to listen on "address:port" and accept new sockets. So this `SO_REUSEPORT` option is trivial in `gnet` but note that it will fall back to the old networking model of `evio` when you enable the `SO_REUSEPORT` option.

Just use functional options to set up `SO_REUSEPORT` and you can enjoy this feature:

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true), gnet.WithReusePort(true)))
```

## Multiple built-in codecs for TCP stream

There are multiple built-in codecs in `gnet` which allow you to encode/decode frames into/from TCP stream.

So far `gnet` has four kinds of built-in codecs: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, which generally meets most scenarios, but still `gnet` allows users to customize their own codecs in their `gnet` servers by implementing the interface [gnet.ICodec](https://github.com/panjf2000/gnet/blob/master/codec.go#L17) and replacing the default codec in `gnet` with customized codec via functional options.

Here is an [example](https://github.com/panjf2000/gnet/blob/master/examples/codec/server/server.go) with codec, showing you how to leverage codec to encode/decode network frames into/from TCP stream.

# 📊 Performance

## Contrasts to the similar networking libraries

## On Linux (epoll)

### Test Environment

```powershell
# Machine information
        OS : Ubuntu 18.04/x86_64
       CPU : 8 Virtual CPUs
    Memory : 16.0 GiB

# Go version and configurations
Go Version : go1.12.9 linux/amd64
GOMAXPROCS=8
```

### 

#### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_linux.png)

#### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_linux.png)

## On FreeBSD (kqueue)

### Test Environment

```powershell
# Machine information
        OS : macOS Mojave 10.14.6/x86_64
       CPU : 4 CPUs
    Memory : 8.0 GiB

# Go version and configurations
Go Version : go version go1.12.9 darwin/amd64
GOMAXPROCS=4
```

#### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_mac.png)

#### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_mac.png)

# 📄 License

Source code in `gnet` is available under the MIT [License](/LICENSE).

# 👏 Contributors

Please read our [Contributing Guidelines](CONTRIBUTING.md) before opening a PR and thank you to all the developers who already made contributions to `gnet`!

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# 🙏 Acknowledgments

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [goframe](https://github.com/smallnest/goframe)

# 📚 Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoll I/O 多路复用构建原生网络模型之源码深度解析](https://taohuawu.club/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://taohuawu.club/go-event-loop-networking-library-gnet)

## JetBrains OS licenses

`gnet` had been being developed with `GoLand` IDE under the **free JetBrains Open Source license(s)** granted by JetBrains s.r.o., hence I would like to express my thanks here.

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>

# 💰 Backers

Support us with a monthly donation and help us continue our activities.

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg"></a>

# 💎 Sponsors

Become a bronze sponsor with a monthly donation of $10 and get your logo on our README on Github.

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg"></a>

# ☕️ Buy me a coffee

<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" align="middle"/></a>&nbsp;&nbsp;