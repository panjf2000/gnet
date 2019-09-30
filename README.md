<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/gnet/master/logo.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Doc for gnet" target="_blank" href="https://gowalker.org/github.com/panjf2000/gnet?lang=en-US"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
</p>

# [[中文](README_ZH.md)]

`gnet` is an Event-Loop networking framework that is fast and small. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [libuv](https://github.com/libuv/libuv) and [libevent](https://github.com/libevent/libevent).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking network library written in pure Go which works on transport layer with TCP/UDP/Unix-Socket protocols, so it allows developers to implement their own protocols of application layer upon `gnet` for building  diversified network applications, for instance, you get a HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from project `evio` while having higher performance.**

# Features

- [High-performance](#Performance) Event-Loop under multi-threads/goroutines model
- Built-in load balancing algorithm: Round-Robin
- Concise APIs
- Efficient memory usage: Ring-Buffer
- Supporting multiple protocols: TCP, UDP, and Unix Sockets
- Supporting two event-notification mechanisms: epoll in Linux and kqueue in FreeBSD
- Supporting asynchronous write operation
- Flexible ticker event
- SO_REUSEPORT socket option

# Key Designs

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

You may ask me a question: what if my business logic in `EventHandler.React`  contains some blocking code which leads to a blocking in event-loop of `gnet`, what is the solution for this kind of situation？

As you know, there is a most important tenet when writing code under `gnet`: you should never block the event-loop in the `EventHandler.React`, otherwise it will lead to a low throughput in your `gnet` server, which is also the most important tenet in `netty`. 

And the solution for that would be found in the subsequent multiple-threads/goroutines model of `gnet`: 『Multiple Reactors with thread/goroutine pool』which pulls you out from the blocking mire, it will construct a worker-pool with fixed capacity and put those blocking jobs in `EventHandler.React` into the worker-pool to unblock the event-loop goroutines.

This new networking model is under development and about to be delivered soon and its architecture diagram of new model is in here:

<p align="center">
<img width="854" alt="multi_reactor_thread_pool" src="https://user-images.githubusercontent.com/7496278/64918783-90de3b80-d7d5-11e9-9190-ff8277c95db1.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img width="916" alt="multi-reactors" src="https://user-images.githubusercontent.com/7496278/64918646-a7839300-d7d3-11e9-804a-d021ddd23ca3.png">
</p>

Before you can benefit from this new networking model in handling blocking business logic, there is still a way for you to handle your business logic in networking: you can utilize the open-source goroutine-pool to unblock your blocking code, and I now present you [ants](https://github.com/panjf2000/ants): a high-performance goroutine pool in Go that allows you to manage and recycle a massive number of goroutines in your concurrency programs.

You can import `ants` to your `gnet` server and put your blocking code to the `ants` pool in `EventHandler.React`, which makes your business code non-blocking.

## Auto-scaling Ring Buffer

`gnet` utilizes Ring-Buffer to cache TCP streams and manage memory cache in networking.

<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916810-4f8b6300-d7b8-11e9-9459-5517760da738.gif">
</p>


# Getting Started

## Installation

```sh
$ go get -u github.com/panjf2000/gnet
```

## Usage

It is easy to create a network server with `gnet`. All you have to do is just make your implementaion of `gnet.EventHandler` interface and register your event-handler functions to it, then pass it to the `gnet.Serve` function along with the binding address(es). Each connection is represented as a `gnet.Conn` interface that is passed to various events to differentiate the clients. At any point you can close a client or shutdown the server by return a `Close` or `Shutdown` action from an event.

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
	top, tail := c.ReadPair()
	out = top
	if tail != nil {
		out = append(top, tail...)
	}
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

	"github.com/panjf2000/ants/v2"
	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
	pool *ants.Pool
}

func (es *echoServer) React(c gnet.Conn) (out []byte, action gnet.Action) {
	data := c.ReadBytes()
	c.ResetBuffer()

	// Use ants pool to unblock the event-loop.
	_ = es.pool.Submit(func() {
		time.Sleep(1 * time.Second)
		c.AsyncWrite(data)
	})

	action = gnet.DataRead
	return
}

func main() {
	// Create a goroutine pool.
	poolSize := 64 * 1024
	pool, _ := ants.NewPool(poolSize, ants.WithNonblocking(true))
	defer pool.Release()
	echo := &echoServer{pool: pool}
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```

Like I said in the 『Multiple Reactors + Goroutine-Pool Model』section, if there are blocking code in your business logic, then you ought to turn them into non-blocking code in any way, for instance you can wrap them into a goroutine, but it will result in a massive amount of goroutines if massive traffic is passing through your server so I would suggest you utilize a goroutine pool like `ants` to manage those goroutines and reduce the cost of system resource.

### I/O Events

Current supported I/O events in `gnet`:

- `EventHandler.OnInitComplete` is activated when the server is ready to accept new connections.
- `EventHandler.OnOpened` is activated when a connection has opened.
- `EventHandler.OnClosed` is activated when a connection has closed.
- `EventHandler.React` is activated when the server receives new data from a connection.
- `EventHandler.Tick` is activated immediately after the server starts and will fire again after a specified interval.
- `EventHandler.PreWrite` is activated just before any data is written to any client socket.


### Ticker

The `EventHandler.Tick` event fires ticks at a specified interval. 
The first tick fires immediately after the `Serving` events and if you intend to set up a ticker event, remember to pass a option: `gnet.WithTicker(true)` to `gnet.Serve`.

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

The `gnet.WithMulticore(true)` indicates whether the server will be effectively created with multi-cores, if so, then you must take care with synchronizing memory between all event callbacks, otherwise, it will run the server with single thread. The number of threads in the server will be automatically assigned to the value of `runtime.NumCPU()`.

## Load balancing

The current built-in load balancing algorithm in `gnet` is Round-Robin.

## SO_REUSEPORT

Servers can utilize the [SO_REUSEPORT](https://lwn.net/Articles/542629/) option which allows multiple sockets on the same host to bind to the same port and the OS kernel takes care of the load balancing for you, it wakes one socket per `accpet` event coming to resolved the `thundering herd`.

Just use functional options to set up `SO_REUSEPORT` and you can enjoy this feature:

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true)))
```

# Performance

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

# License

Source code in `gnet` is available under the MIT [License](/LICENSE).

# Thanks

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)

# Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://taohuawu.club/go-event-loop-networking-library-gnet)
