---
last_modified_on: "2020-07-09"
id: overview
title: "Overview"
description: "High-level description of the gnet framework and its features."
---

## What is gnet?

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv), which makes `gnet` achieve a much higher performance than Go [net](https://golang.org/pkg/net/).

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

## Features

- [x] [High-performance](#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Lock-free during the entire runtime
- [x] Concise and easy-to-use APIs
- [x] Efficient, reusable and elastic memory buffers: Ring-Buffer, Linked-List-Buffer, Mixed-Buffer which combines the first two
- [x] Supporting multiple protocols/IPC mechanism: `TCP`, `UDP` and `Unix Domain Socket`
- [x] Supporting multiple load-balancing algorithms: `Round-Robin`, `Source-Addr-Hash` and `Least-Connections`
- [x] Supporting two event-driven mechanisms: `epoll` on **Linux** and `kqueue` on **FreeBSD/DragonFly/Darwin**
- [x] Flexible ticker event
- [x] Implementation of `gnet` Client

## Architecture

### Networking Model of Multiple Threads/Goroutines
#### Multiple Reactors

`gnet` redesigns and implements a new built-in networking model of multiple threads/goroutines: 『multiple reactors』 which is also the default networking model of multiple threads in `netty`, Here's the schematic diagram:

<p align="center">
<img alt="multi_reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors.png" />
</p>

and it works as the following sequence diagram:
<p align="center">
<img alt="reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors-sequence-diagram.png" />
</p>

#### Multiple Reactors + Goroutine Pool

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

## Key designs

### Elastic Buffer

#### Elastic Ring Buffer

<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif" />
</p>

#### Elastic Ring & Linked-list Buffer

<p align="center">
<img src="https://img.taohuawu.club/gallery/elastic-buffer.png" />
</p>

There are two buffers inside `gnet`: inbound buffer (elastic-ring-buffer) and outbound buffer (elastic-ring&linked-list-buffer) to buffer and manage inbound/outbound network data, inbound and outbound buffers inside gnet are designed and tuned to reuse memory and be auto-scaling on demand.

The purpose of implementing inbound and outbound buffers in `gnet` is to transfer the logic of buffering and managing network data based on application protocol upon TCP stream from business server to framework and unify the network data buffer, which minimizes the complexity of business code so that developers are able to concentrate on business logic instead of the underlying implementation.