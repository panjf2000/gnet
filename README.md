<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square&logo=travis-ci&logoColor=white"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square&logo=codecov"></a>
<a title="Supported Platforms" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/platform-Linux%20%7C%20FreeBSD%20%7C%20DragonFly%20%7C%20Darwin%20%7C%20Windows-549688?style=flat-square&logo=dlna"></a>
<a title="Require Go Version" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/go-%3E%3D1.9-30dff3?style=flat-square&logo=go"></a>
<br/>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<a title="Doc for gnet" target="_blank" href="https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc"><img src="https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs"></a>
<a title="gnet on Sourcegraph" target="_blank" href="https://sourcegraph.com/github.com/panjf2000/gnet?badge"><img src="https://sourcegraph.com/github.com/panjf2000/gnet/-/badge.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go#networking"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/v/release/panjf2000/gnet.svg?color=161823&style=flat-square&logo=smartthings"></a>
<a title="Tag" target="_blank" href="https://github.com/panjf2000/gnet/tags"><img src="https://img.shields.io/github/v/tag/panjf2000/gnet?color=%23ff8936&logo=fitbit&style=flat-square"></a>
</p>

English | [🇨🇳中文](README_ZH.md)

# 📖 Introduction

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv), which makes `gnet` achieve a much higher performance than Go [net](https://golang.org/pkg/net/).

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

# 🚀 Features

- [x] [High-performance](#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Built-in memory pool with bytes powered by the library [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [x] Lock-free during the entire life cycle
- [x] Concise APIs
- [x] Efficient memory usage: Ring-Buffer
- [x] Supporting multiple protocols/IPC mechanism: `TCP`, `UDP` and `Unix Domain Socket`
- [x] Supporting multiple load-balancing algorithms: `Round-Robin`, `Source Addr Hash` and `Least-Connections`
- [x] Supporting two event-driven mechanisms: `epoll` on **Linux** and `kqueue` on **FreeBSD/DragonFly/Darwin**
- [x] Supporting asynchronous write operation
- [x] Flexible ticker event
- [x] SO_REUSEPORT socket option
- [x] Built-in multiple codecs to encode/decode network frames into/from TCP stream: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://netty.io/4.1/api/io/netty/handler/codec/package-summary.html), also supporting customized codecs
- [x] Supporting Windows platform with ~~event-driven mechanism of IOCP~~ Go stdlib: net
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

### Multiple Reactors + Goroutine Pool

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

`gnet` implements the networking model:『multiple reactors with thread/goroutine pool』by the aid of a high-performance goroutine pool called [ants](https://github.com/panjf2000/ants) that allows you to manage and recycle a massive number of goroutines in your concurrent programs, the full features and usages in `ants` are documented [here](https://pkg.go.dev/github.com/panjf2000/ants/v2?tab=doc).

`gnet` integrates `ants` and provides the `pool.goroutine.Default()` method that you can call to instantiate a `ants` pool where you are able to put your blocking code logic and call the function `gnet.Conn.AsyncWrite([]byte)` to send out data asynchronously after you finish the blocking process and get the output data, which makes the goroutine of event-loop non-blocking.

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

**The detailed documentation is located in here: [docs of gnet](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc), but let's pass through the brief instructions first.**

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

As you can see, this example of echo server only sets up the `EventHandler.React` function where you commonly write your main business code and it will be called once the server receives input data from a client. What you should know is that the input parameter: `frame` is a complete packet which has been decoded by the codec, as a general rule, you should implement the `gnet` [codec interface](https://github.com/panjf2000/gnet/blob/master/codec.go#L18-L24) as the business codec to packet and unpacket TCP stream, but if you don't, your `gnet` server is going to work with the [default codec](https://github.com/panjf2000/gnet/blob/master/codec.go#L53-L63) under the acquiescence, which means all data inculding latest data and previous data in buffer will be stored in the input parameter: `frame` when `EventHandler.React` is being triggered. The output data will be then encoded and sent back to that client by assigning the `out` variable and returning it after your business code finish processing data(in this case, it just echo the data back).

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

Like I said in the 『Multiple Reactors + Goroutine Pool』section, if there are blocking code in your business logic, then you ought to turn them into non-blocking code in any way, for instance, you can wrap them into a goroutine, but it will result in a massive amount of goroutines if massive traffic is passing through your server so I would suggest you utilize a goroutine pool like [ants](https://github.com/panjf2000/ants) to manage those goroutines and reduce the cost of system resources.

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
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
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
	var multicore, reuseport bool

	// Example command: go run echo.go --port 9000 --multicore=true --reuseport=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.BoolVar(&reuseport, "reuseport", false, "--reuseport true")
	flag.Parse()
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport)))
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
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
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
	<summary> UDS Echo Server </summary>

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
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
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
	var addr string
	var multicore bool

	// Example command: go run echo.go --sock echo.sock --multicore=true
	flag.StringVar(&addr, "sock", "echo.sock", "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.Parse()

	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("unix://%s", addr), gnet.WithMulticore(multicore)))
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
	"unsafe"

	"github.com/panjf2000/gnet"
)

var res string

type request struct {
	proto, method string
	path, query   string
	head, body    string
	remoteAddr    string
}

type httpServer struct {
	*gnet.EventServer
}

var (
	errMsg      = "Internal Server Error"
	errMsgBytes = []byte(errMsg)
)

type httpCodec struct {
	req request
}

func (hc *httpCodec) Encode(c gnet.Conn, buf []byte) (out []byte, err error) {
	if c.Context() == nil {
		return buf, nil
	}
	return appendResp(out, "500 Error", "", errMsg+"\n"), nil
}

func (hc *httpCodec) Decode(c gnet.Conn) (out []byte, err error) {
	buf := c.Read()
	c.ResetBuffer()

	// process the pipeline
	var leftover []byte
pipeline:
	leftover, err = parseReq(buf, &hc.req)
	// bad thing happened
	if err != nil {
		c.SetContext(err)
		return nil, err
	} else if len(leftover) == len(buf) {
		// request not ready, yet
		return
	}
	out = appendHandle(out, res)
	buf = leftover
	goto pipeline
}

func (hs *httpServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("HTTP server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (hs *httpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if c.Context() != nil {
		// bad thing happened
		out = errMsgBytes
		action = gnet.Close
		return
	}
	// handle the request
	out = frame
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

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// parseReq is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parseReq(data []byte, req *request) (leftover []byte, err error) {
	sdata := b2s(data)
	var i, s int
	var head string
	var clen int
	q := -1
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
		"pushing data every %s ...\n", srv.Addr.String(), srv.Multicore, srv.NumEventLoop, ps.tick.String())
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
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
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


<details>
	<summary> Custom Codec Demo with Client/Server </summary>

**protocol intro:**

```go
// CustomLengthFieldProtocol : custom protocol
// custom protocol header contains Version, ActionType and DataLength fields
// its payload is Data field
type CustomLengthFieldProtocol struct {
	Version    uint16
	ActionType uint16
	DataLength uint32
	Data       []byte
}


// Encode ...
func (cc *CustomLengthFieldProtocol) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	result := make([]byte, 0)

	buffer := bytes.NewBuffer(result)

	// take out the param that `React()` event saved.
	item := c.Context().(CustomLengthFieldProtocol)


	if err := binary.Write(buffer, binary.BigEndian, item.Version); err != nil {
		s := fmt.Sprintf("Pack version error , %v", err)
		return nil, errors.New(s)
	}

	if err := binary.Write(buffer, binary.BigEndian, item.ActionType); err != nil {
		s := fmt.Sprintf("Pack type error , %v", err)
		return nil, errors.New(s)
	}
	dataLen := uint32(len(buf))
	if err := binary.Write(buffer, binary.BigEndian, dataLen); err != nil {
		s := fmt.Sprintf("Pack datalength error , %v", err)
		return nil, errors.New(s)
	}
	if dataLen > 0 {
		if err := binary.Write(buffer, binary.BigEndian, buf); err != nil {
			s := fmt.Sprintf("Pack data error , %v", err)
			return nil, errors.New(s)
		}
	}

	return buffer.Bytes(), nil
}

// Decode ...
func (cc *CustomLengthFieldProtocol) Decode(c gnet.Conn) ([]byte, error) {
	// parse header
	headerLen := DefaultHeadLength // uint16+uint16+uint32
	if size, header := c.ReadN(headerLen); size == headerLen {
		byteBuffer := bytes.NewBuffer(header)
		var pbVersion, actionType uint16
		var dataLength uint32
		binary.Read(byteBuffer, binary.BigEndian, &pbVersion)
		binary.Read(byteBuffer, binary.BigEndian, &actionType)
		binary.Read(byteBuffer, binary.BigEndian, &dataLength)
		// to check the protocol version and actionType,
		// reset buffer if the version or actionType is not correct
		if pbVersion != DefaultProtocolVersion || isCorrectAction(actionType) == false {
			c.ResetBuffer()
			log.Println("not normal protocol:", pbVersion, DefaultProtocolVersion, actionType, dataLength)
			return nil, errors.New("not normal protocol")
		}
		// parse payload
		dataLen := int(dataLength) // max int32 can contain 210MB payload
		protocolLen := headerLen + dataLen
		if dataSize, data := c.ReadN(protocolLen); dataSize == protocolLen {
			c.ShiftN(protocolLen)
			// log.Println("parse success:", data, dataSize)

			// return the payload of the data
			return data[headerLen:], nil
		}
		// log.Println("not enough payload data:", dataLen, protocolLen, dataSize)
		return nil, errors.New("not enough payload data")

	}
	// log.Println("not enough header data:", size)
	return nil, errors.New("not enough header data")
}
```

**Client/Server:**
[Check out the source code](https://github.com/gnet-io/gnet-examples/tree/master/examples/custom_codec).
</details>

**For more details, check out here: [all examples of gnet](https://github.com/gnet-io/gnet-examples/tree/master/examples).**

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

`gnet` supports UDP protocol so the `gnet.Serve` method can bind to UDP addresses. 

- All incoming and outgoing packets will not be buffered but read and sent directly.
- The `EventHandler.OnOpened` and `EventHandler.OnClosed` events are not available for UDP sockets, only the `React` event.
- The UDP equivalents of  `AsyncWrite([]byte)` in TCP is `SendTo([]byte)`.

## Unix Domain Socket

`gnet` also supports UDS(Unix Domain Socket), just pass the UDS addresses like "unix://xxx" to the `gnet.Serve` method and you could play with it.

It is nothing different from making use of TCP when doing stuff with UDS, so the `gnet` UDS servers are able to leverage all event functions which are available under TCP protocol.

## Multi-threads

The `gnet.WithMulticore(true)` indicates whether the server will be effectively created with multi-cores, if so, then you must take care of synchronizing memory between all event callbacks, otherwise, it will run the server with a single thread. The number of threads in the server will be automatically assigned to the value of `runtime.NumCPU()`.

## Load Balancing

`gnet` currently supports three load balancing algorithms: `Round-Robin`, `Source Addr Hash` and `Least-Connections`, you are able to decide which algorithm to use by passing the functional option `LB` (RoundRobin/LeastConnections/SourceAddrHash) to `gnet.Serve`.

If the load balancing algorithm is not specified explicitly, `gnet` will use `Round-Robin` by default.

## SO_REUSEPORT

`gnet` server is able to utilize the [SO_REUSEPORT](https://lwn.net/Articles/542629/) option which allows multiple sockets on the same host to bind to the same port and the OS kernel takes care of the load balancing for you, it wakes one socket per `connect` event coming to resolved the `thundering herd`.

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

## Benchmarks on TechEmpower

```powershell
# Hardware
CPU: 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
Mem: 32GB RAM
OS : Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
Net: Switched 10-gigabit ethernet
Go : go1.14.x linux/amd64
```

![All language](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-all.jpg)

This is the top 50 on the framework ranking of all programming languages consists of a total of 382 frameworks from all over the world.


![Golang](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-go.png)

This is the full framework ranking of Golang.

To see the full ranking list, visit [Full ranking list of Plaintext](https://www.techempower.com/benchmarks/#section=test&runid=c7152e8f-5b33-4ae7-9e89-630af44bc8de&hw=ph&test=plaintext).

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

Please read the [Contributing Guidelines](CONTRIBUTING.md) before opening a PR and thank you to all the developers who already made contributions to `gnet`!

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# 🙏 Acknowledgments

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [go_reuseport](https://github.com/kavu/go_reuseport)
- [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [goframe](https://github.com/smallnest/goframe)
- [ringbuffer](https://github.com/smallnest/ringbuffer)

# 📚 Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoll I/O 多路复用构建原生网络模型之源码深度解析](https://taohuawu.club/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://taohuawu.club/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://taohuawu.club/releasing-gnet-v1-with-techempower)
- [字节跳动在 Go 网络库上的实践](https://taohuawu.club/bytedance-network-library-practices)

# 🖥 User cases

Please feel free to add your projects here~~

<!--
<a href="https://github.com/panjf2000/gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" width="150" align="middle"/></a>&nbsp;&nbsp;
<a href="https://www.tencent.com"><img src="https://www.tencent.com/img/index/tencent_logo.png" width="250" align="middle"/></a>&nbsp;&nbsp;
-->

# 🔋 JetBrains OS licenses

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