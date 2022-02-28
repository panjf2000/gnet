---
id: doc-for-gnet-v1
last_modified_on: "2022-02-26"
title: Doc for gnet v1
description: "Hello World! We present you, gnet!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

# 🎉 Getting Started

## Prerequisites

`gnet` requires Go 1.9 or later.

## Installation

```powershell
go get -u github.com/panjf2000/gnet
```

`gnet` is available as a Go module, with [Go 1.11 Modules](https://github.com/golang/go/wiki/Modules) support (Go 1.11+), just simply `import "github.com/panjf2000/gnet"` in your source code and `go [build|run|test]` will download the necessary dependencies automatically.

## Usage Examples

**The detailed documentation is located here: [docs of gnet](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc), but let's pass through the brief instructions first.**

It is easy to create a network server with `gnet`. All you have to do is just to make your implementation of `gnet.EventHandler` interface and register your event-handler functions to it, then pass it to the `gnet.Serve` function along with the binding address(es). Each connection is represented as a `gnet.Conn` interface that is passed to various events to differentiate the clients. At any point you can close a connection or shutdown the server by return a `Close` or `Shutdown` action from an event function.

The simplest example to get you started playing with `gnet` would be the echo server. So here you are, a simplest echo server upon `gnet` that is listening on port 9000:

### Echo server without blocking logic

<details>
	<summary> Old version(&lt;=v1.0.0-rc.4)  </summary>

```go
package main

import (
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	gnet.EventServer
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
	gnet.EventServer
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

As you can see, this example of echo server only sets up the `EventHandler.React` function where you commonly write your main business code and it will be called once the server receives input data from a client. What you should know is that the input parameter: `frame` is a complete packet which has been decoded by the codec, as a general rule, you should implement the `gnet` [codec interface](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#ICodec) as the business codec to packet and unpacket TCP stream, but if you don't, your `gnet` server is going to work with the [default codec](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#BuiltInFrameCodec) under the acquiescence, which means all data inculding latest data and previous data in buffer will be stored in the input parameter: `frame` when `EventHandler.React` is being triggered. The output data will be then encoded and sent back to that client by assigning the `out` variable and returning it after your business code finish processing data(in this case, it just echo the data back).

### Echo server with blocking logic

<details>
	<summary> Old version(&lt;=v1.0.0-rc.4)  </summary>

```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type echoServer struct {
	gnet.EventServer
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
	gnet.EventServer
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

- [Versions prior to v1.6.0](https://github.com/gnet-io/gnet-examples/tree/984eed8bda54282f0d6cb2005ff48e529ae8980d)

- [Versions of v1.6.x](https://github.com/gnet-io/gnet-examples/tree/master)

- [Versions of v2.x.x](https://github.com/gnet-io/gnet-examples)

## I/O Events

Current supported I/O events in `gnet`:

- `EventHandler.OnInitComplete` fires when the server has been initialized and ready to accept new connections.
- `EventHandler.OnOpened` fires once a connection has been opened.
- `EventHandler.OnClosed` fires after a connection has been closed.
- `EventHandler.React` fires when the server receives inbound data from a socket/connection. (usually it is where you write the code of business logic)
- `EventHandler.Tick` fires right after the server starts and then fires every specified interval.
- `EventHandler.PreWrite` fires just before any data has been written to client.

## poll_opt mode

By default, `gnet` utilizes the standard package `golang.org/x/sys/unix` to implement pollers with `epoll` or `kqueue`, where a HASH MAP of `fd->conn` is introduced to help retrieve connections by file descriptors returned from pollers, but now the user can run `go build` with build tags `poll_opt`, like this: `go build -tags=poll_opt`, and `gnet` then switch to the optimized implementations of pollers that invoke the system calls of `epoll` or `kqueue` directly and add file descriptors to the interest list along with storing the corresponding connection pointers into `epoll_data` or `kevent`, in which case `gnet` can get rid of the HASH MAP of `fd->conn` and regain each connection pointer by the conversion of `void*` pointer in the I/O event-looping. In theory, it ought to achieve a higher performance with this optimization. 

See [#230](https://github.com/panjf2000/gnet/pull/230) for code details.

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

- All incoming and outgoing packets will not be buffered but read and sent directly, which means all functions of `gnet.Conn` that manipulate the internal buffers are not available; users should use the `frame []byte` from the `gnet.React(frame []byte, c gnet.Conn)` as the UDP packet instead calling functions of `gnet.Conn`, like `c.Read()`, `c.ResetBuffer()`, `c.BufferLength()` and so on, to process data.
- The `EventHandler.OnOpened` and `EventHandler.OnClosed` events are not available for UDP sockets, only the `React` event.
- The UDP equivalents of  `AsyncWrite([]byte)` in TCP is `SendTo([]byte)`.

## Unix Domain Socket

`gnet` also supports UDS(Unix Domain Socket), just pass the UDS addresses like "unix://xxx" to the `gnet.Serve` method and you could play with it.

It is nothing different from making use of TCP when doing stuff with UDS, so the `gnet` UDS servers are able to leverage all event functions which are available under TCP protocol.

## Multi-threads

The `gnet.WithMulticore(true)` indicates whether the server will be effectively created with multi-cores, if so, then you must take care of synchronizing memory between all event callbacks, otherwise, it will run the server with a single thread. The number of threads in the server will be automatically assigned to the value of `runtime.NumCPU()`.

## Load Balancing

`gnet` currently supports three load balancing algorithms: `Round-Robin`, `Source-Addr-Hash` and `Least-Connections`, you are able to decide which algorithm to use by passing the functional option `LB` (RoundRobin/LeastConnections/SourceAddrHash) to `gnet.Serve`.

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

So far `gnet` has four kinds of built-in codecs: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, which generally meets most scenarios, but still `gnet` allows users to customize their own codecs in their `gnet` servers by implementing the interface [gnet.ICodec](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#ICodec) and replacing the default codec in `gnet` with customized codec via functional options.

Here is an [example](https://github.com/gnet-io/gnet-examples/tree/master/examples/codec) with codec, showing you how to leverage codec to encode/decode network frames into/from TCP stream.

# 📊 Performance

## Benchmarks on TechEmpower

```powershell
# Hardware Environment
CPU: 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
Mem: 32GB RAM
OS : Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
Net: Switched 10-gigabit ethernet
Go : go1.14.x linux/amd64
```

![All languages](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-all.jpg)

This is the ***top 50*** on the framework ranking of all programming languages consists of a total of ***422 frameworks*** from all over the world where `gnet` is the ***runner-up***.


![Golang](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-go.png)

This is the full framework ranking of Go and `gnet` tops all the other frameworks, which makes `gnet` the ***fastest*** networking framework in Go.

To see the full ranking list, visit [TechEmpower Plaintext Benchmark](https://www.techempower.com/benchmarks/#section=test&runid=53c6220a-e110-466c-a333-2e879fea21ad&hw=ph&test=plaintext).

## Contrasts to the similar networking libraries

## On Linux (epoll)

### Test Environment

```powershell
# Machine information
        OS : Ubuntu 20.04/x86_64
       CPU : 8 CPU cores, AMD EPYC 7K62 48-Core Processor
    Memory : 16.0 GiB

# Go version and settings
Go Version : go1.17.2 linux/amd64
GOMAXPROCS : 8

# Benchmark parameters
TCP connections : 1000/2000/5000/10000
Packet size     : 512/1024/2048/4096/8192/16384/32768/65536 bytes
Test duration   : 15s
```

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_linux.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_linux.png)

## On MacOS (kqueue)

### Test Environment

```powershell
# Machine information
        OS : MacOS Big Sur/x86_64
       CPU : 6 CPU cores, Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
    Memory : 16.0 GiB

# Go version and settings
Go Version : go1.16.5 darwin/amd64
GOMAXPROCS : 12

# Benchmark parameters
TCP connections : 300/400/500/600/700
Packet size     : 512/1024/2048/4096/8192 bytes
Test duration   : 15s
```

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_macos.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_macos.png)

# ️🚨 License

Source code in `gnet` is available under the [MIT License](https://github.com/panjf2000/gnet/blob/master/LICENSE).

# 👏 Contributors

Please read the [Contributing Guidelines](https://github.com/panjf2000/gnet/blob/master/CONTRIBUTING.md) before opening a PR and thank you to all the developers who already made contributions to `gnet`!

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# 🙏 Acknowledgments

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [go_reuseport](https://github.com/kavu/go_reuseport)
- [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [goframe](https://github.com/smallnest/goframe)
- [ringbuffer](https://github.com/smallnest/ringbuffer)

# ⚓ Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoller 原生网络模型之源码全面揭秘](https://strikefreedom.top/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://strikefreedom.top/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://strikefreedom.top/releasing-gnet-v1-with-techempower)

# 🎡 Use cases

The following companies/organizations use `gnet` as the underlying network service in production.

<a href="https://www.tencent.com"><img src="https://img.taohuawu.club/gallery/tencent_logo.png" width="250" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.iqiyi.com" target="_blank"><img src="https://img.taohuawu.club/gallery/iqiyi-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.mi.com" target="_blank"><img src="https://img.taohuawu.club/gallery/mi-logo.png" width="150" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.360.com" target="_blank"><img src="https://img.taohuawu.club/gallery/360-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://tieba.baidu.com/" target="_blank"><img src="https://img.taohuawu.club/gallery/baidu-tieba-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://game.qq.com/" target="_blank"><img src="https://img.taohuawu.club/gallery/tencent-games-logo.png" width="200" align="middle"/></a>

If your projects are also using `gnet`, feel free to open a [pull request](https://github.com/panjf2000/gnet/pulls) refreshing this list of use cases.

# 💰 Backers

Support us with a monthly donation and help us continue our activities.

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg" /></a>

# 💎 Sponsors

Become a bronze sponsor with a monthly donation of $10 and get your logo on our README on Github.

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg" /></a>

# ☕️ Buy me a coffee

> Please be sure to leave your name, Github account or other social media accounts when you donate by the following means so that I can add it to the list of donors as a token of my appreciation.

<table><tr>
<td><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" /></td>
<td><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" /></td>
<td><a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" /></a></td>
</tr></table>

# 💴 Patrons

<a target="_blank" href="https://github.com/patrick-othmer"><img src="https://avatars1.githubusercontent.com/u/8964313" width="100" alt="Patrick Othmer" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://avatars2.githubusercontent.com/u/50285334" width="100" alt="Jimmy" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/cafra"><img src="https://avatars0.githubusercontent.com/u/13758306" width="100" alt="ChenZhen" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/yangwenmai"><img src="https://avatars0.githubusercontent.com/u/1710912" width="100" alt="Mai Yang" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/BeijingWks"><img src="https://avatars3.githubusercontent.com/u/33656339" width="100" alt="王开帅" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/refs"><img src="https://avatars3.githubusercontent.com/u/6905948" width="100" alt="Unger Alejandro" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/Swaggadan"><img src="https://avatars.githubusercontent.com/u/137142" width="100" alt="Swaggadan" /></a>&nbsp;<a target="_blank" href="https://github.com/Wuvist"><img src="https://avatars.githubusercontent.com/u/657796" width="100" alt="Weng Wei" /></a>

# 🔑 JetBrains OS licenses

`gnet` had been being developed with `GoLand` IDE under the **free JetBrains Open Source license(s)** granted by JetBrains s.r.o., hence I would like to express my thanks here.
<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle" /></a>

# 🔋 Sponsorship

<p>
	<h3>This project is supported by:</h3>
	<a href="https://www.digitalocean.com/"><img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/SVG/DO_Logo_horizontal_blue.svg" width="201px" />
	</a>
</p>