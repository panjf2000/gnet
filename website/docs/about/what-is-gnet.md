---
last_modified_on: "2020-07-09"
title: "What is Gnet?"
description: "High-level description of the Gnet framework and its features."
---

## 📖 Introduction

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv), which makes `gnet` achieve a much higher performance than Go [net](https://golang.org/pkg/net/).

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

## 🚀 Features

- [x] [High-performance](#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Built-in memory pool with bytes powered by the library [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [x] Lock-free during the entire life cycle
- [x] Concise APIs
- [x] Efficient memory usage: Ring-Buffer
- [x] Supporting multiple protocols/IPC mechanism: `TCP`, `UDP` and `Unix Domain Socket`
- [x] Supporting multiple load-balancing algorithms: `Round-Robin`, `Source-Addr-Hash` and `Least-Connections`
- [x] Supporting two event-driven mechanisms: `epoll` on **Linux** and `kqueue` on **FreeBSD/DragonFly/Darwin**
- [x] Supporting asynchronous write operation
- [x] Flexible ticker event
- [x] SO_REUSEPORT socket option
- [x] Built-in multiple codecs to encode/decode network frames into/from TCP stream: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://netty.io/4.1/api/io/netty/handler/codec/package-summary.html), also supporting customized codecs
- [x] Supporting Windows platform with ~~event-driven mechanism of IOCP~~ Go stdlib: net
- [ ] Implementation of `gnet` Client

## 📚 Documentation

For more details about using `gnet`, please visit <a href="https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc" target="_blank">documentations for gnet</a>.