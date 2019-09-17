<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916411-0f27e700-d7b0-11e9-976c-738ff75be8c0.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Godoc for gnet" target="_blank" href="https://godoc.org/github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?style=flat-square"></a>
</p>

# [[中文]](README_ZH.md)

`gnet` is an Event-Loop networking framework that is fast and small. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [libuv](https://github.com/libuv/libuv) and [libevent](https://github.com/libevent/libevent).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling.

`gnet` sells itself as a high-performance, lightweight, nonblocking network library written in pure Go, it works on transport layer with TCP/UDP/Unix-Socket protocols, so it allows developers to implement their own protocols of application layer upon `gnet` for building  diversified network applications, for instance, you get a HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gent` derives from project `evio` while having higher performance.**

# Features

- [High-performance](#Performance) Event-Loop under multi-threads model
- Built-in load balancing algorithm: Round-Robin
- Concise APIs
- Efficient memory usage: Ring-Buffer
- Supporting multiple protocols: TCP, UDP, and Unix Sockets
- Supporting two event-notification mechanisms: epoll in Linux and kqueue in FreeBSD
- Supporting asynchronous write operation
- Allowing multiple network binding on the same Event-Loop
- Flexible ticker event
- SO_REUSEPORT socket option

# Key Designs

## Multiple-threads Model

`gnet` redesigns and implements a new built-in multiple-threads model: 『Multiple Reactors』 which is also the default multiple-threads model of `netty`, Here's the schematic diagram:

<p align="center">
<img width="820" alt="multi_reactor" src="https://user-images.githubusercontent.com/7496278/64916634-8f038080-d7b3-11e9-82c8-f77e9791df86.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img width="869" alt="reactor" src="https://user-images.githubusercontent.com/7496278/64918644-a5213900-d7d3-11e9-88d6-1ec1ec72c1cd.png">
</p>

The subsequent multiple-threads model of `gnet`: 『Multiple Reactors with thread/goroutine pool』is under development and about to be delivered soon, the architecture diagram of new model is in here:

<p align="center">
<img width="854" alt="multi_reactor_thread_pool" src="https://user-images.githubusercontent.com/7496278/64918783-90de3b80-d7d5-11e9-9190-ff8277c95db1.png">
</p>

and it works as the following sequence diagram:
<p align="center">
<img width="916" alt="multi-reactors" src="https://user-images.githubusercontent.com/7496278/64918646-a7839300-d7d3-11e9-804a-d021ddd23ca3.png">
</p>

## Communication Mechanism

`gnet` builds its 『Multiple Reactors』Model under Goroutines in Golang, one Reactor per Goroutine, so there is a critical requirement handling extremely large amounts of messages between Goroutines in this networking model of `gnet`, which means `gnet` needs a efficient communication mechanism between Goroutines. I choose a tricky solution of Disruptor(Ring-Buffer) which provides a higher performance of messages dispatching in networking, instead of the recommended pattern: CSP(Channel) under Golang-Best-Practices.

That is why I finally settle on [go-disruptor](https://github.com/smartystreets-prototypes/go-disruptor): the Golang port of the LMAX Disruptor(a high performance inter-thread messaging library).

## Auto-scaling Ring Buffer

`gnet` leverages Ring-Buffer to cache TCP streams and manage memory cache in networking.

<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916810-4f8b6300-d7b8-11e9-9459-5517760da738.gif">
</p>


# Getting Started

## Installation

```sh
$ go get -u github.com/panjf2000/gnet
```

## Example

```go
// ======================== Echo Server implemented with gnet ===========================

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/ringbuffer"
)

func main() {
	var port int
	var loops int
	var udp bool
	var trace bool
	var reuseport bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.IntVar(&loops, "loops", 0, "num loops")
	flag.Parse()

	var events gnet.Events
	events.NumLoops = loops
	events.OnInitComplete = func(srv gnet.Server) (action gnet.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		return
	}
	events.React = func(c gnet.Conn, inBuf *ringbuffer.RingBuffer) (out []byte, action gnet.Action) {
		top, tail := inBuf.PreReadAll()
		out = append(top, tail...)
		inBuf.Reset()

		if trace {
			log.Printf("%s", strings.TrimSpace(string(top)+string(tail)))
		}
		return
	}
	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	log.Fatal(gnet.Serve(events, fmt.Sprintf("%s://:%d", scheme, port)))
}

```

## I/O Events

Current supported I/O events in `gnet`:

- `OnInitComplete` is activated when the server is ready to accept new connections.
- `OnOpened` is activated when a connection has opened.
- `OnClosed` is activated when a connection has closed.
- `OnDetached` is activated when a connection has been detached using the `Detach` return action.
- `React` is activated when the server receives new data from a connection.
- `Tick` is activated immediately after the server starts and will fire again after a specified interval.
- `PreWrite` is activated just before any data is written to any client socket.

# Performance

## On Linux (epoll)

### Test Environment

```powershell
Go Version : go1.12.9 linux/amd64
        OS : Ubuntu 18.04/x86_64
       CPU : 8 Virtual CPUs
    Memory : 16.0 GiB
```

### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_linux.png)

### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_linux.png)

## On FreeBSD (kqueue)

### Test Environment

```powershell
Go Version : go version go1.12.9 darwin/amd64
        OS : macOS Mojave 10.14.6/x86_64
       CPU : 4 CPUs
    Memory : 8.0 GiB
```

### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_mac.png)

### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_mac.png)

# License

Source code in `gnet` is available under the MIT [License](/LICENSE).

# TODO

> gnet is still under active development so the code and documentation will continue to be updated, if you are interested in gnet, please feel free to make your code contributions to it, also if you like gnet, give it a star ~~