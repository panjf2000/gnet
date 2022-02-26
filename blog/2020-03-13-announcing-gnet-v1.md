---
last_modified_on: "2020-03-13"
id: announcing-gnet-v1
title: Announcing gnet v1.0.0
description: "Hello World! We present you, gnet!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

## Today, we are release gnet v1.0.0, enjoy this ultra-fast framework of networking!

<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet" />
</p>

# 📖 Introduction

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv), which makes `gnet` achieve a much higher performance than Go [net](https://golang.org/pkg/net/).

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

# 🚀 Features

- [x] [High-performance](https://github.com/panjf2000/gnet/blob/v1.0.0/README.md#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in load balancing algorithm: Round-Robin
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Built-in memory pool with bytes powered by the library [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [x] Concise APIs
- [x] Efficient memory usage: Ring-Buffer
- [x] Supporting multiple protocols/IPC mechanism: TCP, UDP and Unix Domain Socket
- [x] Supporting two event-driven mechanisms: epoll on Linux and kqueue on FreeBSD
- [x] Supporting asynchronous write operation
- [x] Flexible ticker event
- [x] SO_REUSEPORT socket option
- [x] Built-in multiple codecs to encode/decode network frames into/from TCP stream: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://netty.io/4.1/api/io/netty/handler/codec/package-summary.html), also supporting customized codecs
- [x] Supporting Windows platform with ~~event-driven mechanism of IOCP~~ Go stdlib: net
- [ ] Additional load-balancing algorithms: Random, Least-Connections, Consistent-hashing and so on
- [ ] TLS support
- [ ] Implementation of `gnet` Client

# 💡 Key Designs

## Networking Model of Multiple Threads/Goroutines
### Multiple Reactors

`gnet` redesigns and implements a new built-in networking model of multiple threads/goroutines: 『multiple reactors』 which is also the default networking model of multiple threads in `netty`, Here's the schematic diagram:

<p align="center">
<img alt="multi_reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors.png" />
</p>

and it works as the following sequence diagram:
<p align="center">
<img alt="reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors-sequence-diagram.png" />
</p>

### Multiple Reactors + Goroutine Pool

You may ask me a question: what if my business logic in `EventHandler.React`  contains some blocking code which leads to blocking in event-loop of `gnet`, what is the solution for this kind of situation？

As you know, there is a most important tenet when writing code under `gnet`: you should never block the event-loop goroutine in the `EventHandler.React`, which is also the most important tenet in `netty`, otherwise, it will result in a low throughput in your `gnet` server.

And the solution to that could be found in the subsequent networking model of multiple threads/goroutines in `gnet`: 『multiple reactors with thread/goroutine pool』which pulls you out from the blocking mire, it will construct a worker-pool with fixed capacity and put those blocking jobs in `EventHandler.React` into the worker-pool to make the event-loop goroutines non-blocking.

The networking model:『multiple reactors with thread/goroutine pool』dissolves the blocking jobs by introducing a goroutine pool, as shown below:

<p align="center">
<img alt="multi_reactor_thread_pool" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool.png" />
</p>

and it works as the following sequence diagram:
<p align="center">
<img alt="multi-reactors" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool-sequence-diagram.png" />
</p>

`gnet` implements the networking model:『multiple reactors with thread/goroutine pool』by the aid of a high-performance goroutine pool called [ants](https://github.com/panjf2000/ants) that allows you to manage and recycle a massive number of goroutines in your concurrent programs, the full features and usages in `ants` are documented [here](https://pkg.go.dev/github.com/panjf2000/ants/v2?tab=doc).

`gnet` integrates `ants` and provides the `pool.goroutine.Default()` method that you can call to instantiate a `ants` pool where you are able to put your blocking code logic and call the function `gnet.Conn.AsyncWrite([]byte)` to send out data asynchronously after you finish the blocking process and get the output data, which makes the goroutine of event-loop non-blocking.

The details about integrating `gnet`  with `ants` are shown [here](#echo-server-with-blocking-logic).

## Reusable and auto-scaling Ring Buffer

There are two ring-buffers inside `gnet`: inbound buffer and outbound buffer to buffer and manage inbound/outbound network data, ring-buffer inside gnet is designed and tuned to reuse memory and be auto-scaling on demand.

The purpose of implementing inbound and outbound ring-buffers in `gnet` is to transfer the logic of buffering and managing network data based on application protocol upon TCP stream from business server to framework and unify the network data buffer, which minimizes the complexity of business code so that developers are able to concentrate on business logic instead of the underlying implementation.

<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif" />
</p>
