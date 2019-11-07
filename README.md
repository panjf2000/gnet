<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" width="300" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<a title="gnet on Sourcegraph" target="_blank" href="https://sourcegraph.com/github.com/panjf2000/gnet?badge"><img src="https://sourcegraph.com/github.com/panjf2000/gnet/-/badge.svg?style=flat-square"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Doc for gnet" target="_blank" href="https://gowalker.org/github.com/panjf2000/gnet?lang=en-US"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
</p>

English | [🇨🇳中文](README_ZH.md)

# 📖 Introduction

`gnet` is an event-driven networking framework that is fast and small. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP/Unix-Socket protocols, so it allows developers to implement their own protocols of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance.**

# 🚀 Features

- [x] [High-performance](#-performance) event-loop under multi-threads/goroutines model
- [x] Built-in load balancing algorithm: Round-Robin
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Built-in memory pool with bytes powered by the library [pool](https://github.com/gobwas/pool/)
- [x] Concise APIs
- [x] Efficient memory usage: Ring-Buffer
- [x] Supporting multiple protocols: TCP, UDP, and Unix Sockets
- [x] Supporting two event-notification mechanisms: epoll on Linux and kqueue on FreeBSD
- [x] Supporting asynchronous write operation
- [x] Flexible ticker event
- [x] SO_REUSEPORT socket option
- [x] Codec implementations to encode/decode TCP stream to frame: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://github.com/netty/netty/tree/4.1/codec/src/main/java/io/netty/handler/codec)
- [ ] Additional load-balancing algorithms: Random, Least-Connections, Consistent-hashing and so on
- [ ] New event-notification mechanism: IOCP on Windows platform 
- [ ] TLS support
- [ ] Implementation of `gnet` Client

# 💡 Key Designs

## Multiple-Threads/Goroutines Model
### Multiple Reactors Model

`gnet` redesigns and implements a new built-in multiple-threads/goroutines model: 『Multiple Reactors』 which is also the default multiple-threads model of `netty`, Here's the schematic diagram:

<p align="center">
<img width="820" alt="multi_reactor" src="https://user-images.githubusercontent.com/7496278/64916634-8f038080-d7b3-11e9-82c8-f77e9791df86.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img width="869" alt="reactor" src="https://user-images.githubusercontent.com/7496278/64918644-a5213900-d7d3-11e9-88d6-1ec1ec72c1cd.png">
</p>

### Multiple Reactors + Goroutine-Pool Model

You may ask me a question: what if my business logic in `EventHandler.React`  contains some blocking code which leads to blocking in event-loop of `gnet`, what is the solution for this kind of situation？

As you know, there is a most important tenet when writing code under `gnet`: you should never block the event-loop in the `EventHandler.React`, otherwise, it will lead to a low throughput in your `gnet` server, which is also the most important tenet in `netty`. 

And the solution for that could be found in the subsequent multiple-threads/goroutines model of `gnet`: 『Multiple Reactors with thread/goroutine pool』which pulls you out from the blocking mire, it will construct a worker-pool with fixed capacity and put those blocking jobs in `EventHandler.React` into the worker-pool to make the event-loop goroutines non-blocking.

The architecture diagram of『Multiple Reactors with thread/goroutine pool』networking model architecture is in here:

<p align="center">
<img width="854" alt="multi_reactor_thread_pool" src="https://user-images.githubusercontent.com/7496278/64918783-90de3b80-d7d5-11e9-9190-ff8277c95db1.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img width="916" alt="multi-reactors" src="https://user-images.githubusercontent.com/7496278/64918646-a7839300-d7d3-11e9-804a-d021ddd23ca3.png">
</p>

`gnet` implements the networking model of 『Multiple Reactors with thread/goroutine pool』by the aid of a high-performance goroutine pool called [ants](https://github.com/panjf2000/ants) that allows you to manage and recycle a massive number of goroutines in your concurrent programs, the full features and usages in `ants` are documented [here](https://gowalker.org/github.com/panjf2000/ants?lang=en-US).

`gnet` integrates `ants` and provides the `pool.NewWorkerPool` method that you can invoke to instantiate a `ants` pool where you are able to put your blocking code logic in `EventHandler.React` and invoke the function of `gnet.Conn.AsyncWrite` to send out data asynchronously in worker pool after you finish the blocking process and get the output data, which makes the goroutine of event-loop non-blocking.

The details about integrating `gnet`  with `ants` are shown [here](#echo-server-with-blocking-logic).

## Auto-scaling Ring Buffer

`gnet` utilizes Ring-Buffer to buffer network data and manage memories in networking.

<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916810-4f8b6300-d7b8-11e9-9459-5517760da738.gif">
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

It is easy to create a network server with `gnet`. All you have to do is just make your implementation of `gnet.EventHandler` interface and register your event-handler functions to it, then pass it to the `gnet.Serve` function along with the binding address(es). Each connection is represented as a `gnet.Conn` interface that is passed to various events to differentiate the clients. At any point you can close a client or shutdown the server by return a `Close` or `Shutdown` action from an event.

The simplest example to get you started playing with `gnet` would be the echo server. So here you are, a simplest echo server upon `gnet` that is listening on port 9000:

### Echo server without blocking logic

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

As you can see, this example of echo server only sets up the `EventHandler.React` function where you commonly write your main business code and it will be invoked once the server receives input data from a client. The output data will be then sent back to that client by assigning the `out` variable and return it after your business code finish processing data(in this case, it just echo the data back).

### Echo server with blocking logic

```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool"
)

type echoServer struct {
	*gnet.EventServer
	pool *pool.WorkerPool
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
	p := pool.NewWorkerPool()
	defer p.Release()
	
	echo := &echoServer{pool: p}
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```

Like I said in the 『Multiple Reactors + Goroutine-Pool Model』section, if there are blocking code in your business logic, then you ought to turn them into non-blocking code in any way, for instance you can wrap them into a goroutine, but it will result in a massive amount of goroutines if massive traffic is passing through your server so I would suggest you utilize a goroutine pool like `ants` to manage those goroutines and reduce the cost of system resources.

**For more examples, check out here: [examples of gnet](https://github.com/panjf2000/gnet/tree/master/examples).**

## I/O Events

Current supported I/O events in `gnet`:

- `EventHandler.OnInitComplete` is activated when the server is ready to accept new connections.
- `EventHandler.OnOpened` is activated when a connection has opened.
- `EventHandler.OnClosed` is activated when a connection has closed.
- `EventHandler.React` is activated when the server receives new data from a connection. (usually it is where you write the code of business logic)
- `EventHandler.Tick` is activated immediately after the server starts and will fire again after a specified interval.
- `EventHandler.PreWrite` is activated just before any data is written to any client socket.


## Ticker

The `EventHandler.Tick` event fires ticks at a specified interval. 
The first tick fires immediately after the `Serving` events and if you intend to set up a ticker event, remember to pass an option: `gnet.WithTicker(true)` to `gnet.Serve`.

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

## Multi-threads

The `gnet.WithMulticore(true)` indicates whether the server will be effectively created with multi-cores, if so, then you must take care of synchronizing memory between all event callbacks, otherwise, it will run the server with a single thread. The number of threads in the server will be automatically assigned to the value of `runtime.NumCPU()`.

## Load balancing

The current built-in load balancing algorithm in `gnet` is Round-Robin.

## SO_REUSEPORT

Servers can utilize the [SO_REUSEPORT](https://lwn.net/Articles/542629/) option which allows multiple sockets on the same host to bind to the same port and the OS kernel takes care of the load balancing for you, it wakes one socket per `accpet` event coming to resolved the `thundering herd`.

Just use functional options to set up `SO_REUSEPORT` and you can enjoy this feature:

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true), gnet.WithReusePort(true)))
```

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

# 🙏 Thanks

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [pool](https://github.com/gobwas/pool)
- [goframe](https://github.com/smallnest/goframe)

# 📚 Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
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