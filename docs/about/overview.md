---
last_modified_on: "2024-03-04"
id: overview
title: "Overview"
description: "High-level description of the gnet framework and its features."
---

## What is gnet?

`gnet` is an event-driven networking framework that is ultra-fast and lightweight. It is built from scratch by exploiting [epoll](https://man7.org/linux/man-pages/man7/epoll.7.html) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) and it can achieve much higher performance with lower memory consumption than Go [net](https://golang.org/pkg/net/) in many specific scenarios.

`gnet` and [net](https://golang.org/pkg/net/) don't share the same philosophy about network programming. Thus, building network applications with `gnet` can be significantly different from building them with [net](https://golang.org/pkg/net/), and the philosophies can't be harmonized. There are other similar products written in other programming languages in the community, such as [libevent](https://github.com/libevent/libevent), [libuv](https://github.com/libuv/libuv), [netty](https://github.com/netty/netty), [twisted](https://github.com/twisted/twisted), [tornado](https://github.com/tornadoweb/tornado), etc. which work in a similar pattern as `gnet` under the hood.

`gnet` is not designed to displace the Go [net](https://golang.org/pkg/net/), but to create an alternative in the Go ecosystem for building performance-sensitive network services. As a result of which, `gnet` is not as comprehensive as Go [net](https://golang.org/pkg/net/), it provides only the core functionalities (in a concise API set) required by a network application and it is not planned on being a coverall networking framework, as I think [net](https://golang.org/pkg/net/) has done a good enough job in that area.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on the transport layer with TCP/UDP protocols and Unix Domain Socket. It enables developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building diversified network services. For instance, you get an HTTP Server if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

## Features

- [x] [High-performance](#-performance) event-driven looping based on a networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Lock-free during the entire runtime
- [x] Concise and easy-to-use APIs
- [x] Efficient, reusable, and elastic memory buffer: (Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] Multiple protocols/IPC mechanisms: `TCP`, `UDP`, and `Unix Domain Socket`
- [x] Multiple load-balancing algorithms: `Round-Robin`, `Source-Addr-Hash`, and `Least-Connections`
- [x] Two event-driven mechanisms: `epoll` on **Linux** and `kqueue` on **FreeBSD/DragonFly/Darwin**
- [x] Flexible ticker event
- [x] Implementation of `gnet` Client
- [x] **Windows** platform support (For compatibility in development only, do not use it in production)
- [ ] Multiple network addresses binding
- [ ] **TLS** support
- [ ] [io_uring](https://kernel.dk/io_uring.pdf) support

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

You may ask me a question: what if my business logic in `EventHandler.OnTraffic` contains some blocking code which leads to blocking in event-loop of `gnet`, what is the solution for this kind of situation？

As you know, there is a most important tenet when writing code under `gnet`: you should never block the event-loop goroutine in the `EventHandler.OnTraffic`, which is also the most important tenet in `netty`, otherwise, it will result in a low throughput in your `gnet` server.

And the solution to that could be found in the subsequent networking model of multiple threads/goroutines in `gnet`: 『multiple reactors with thread/goroutine pool』which pulls you out from the blocking mire, it will construct a worker-pool with fixed capacity and put those blocking jobs in `EventHandler.OnTraffic` into the worker-pool to make the event-loop goroutines non-blocking.

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
<img src="https://res.strikefreedom.top/static_res/blog/figures/elastic-buffer.png" />
</p>

There are two buffers inside `gnet`: inbound buffer (elastic-ring-buffer) and outbound buffer (elastic-ring&linked-list-buffer) to buffer and manage inbound/outbound network data, inbound and outbound buffers inside gnet are designed and tuned to reuse memory and be auto-scaling on demand.

The purpose of implementing inbound and outbound buffers in `gnet` is to transfer the logic of buffering and managing network data based on application protocol upon TCP stream from business server to framework and unify the network data buffer, which minimizes the complexity of business code so that developers are able to concentrate on business logic instead of the underlying implementation.