---
last_modified_on: "2020-07-10"
title: "Gnet 是什么?"
description: "宏观角度陈述 gnet 框架。"
---

## 📖 简介

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。它直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Go 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[netty](https://github.com/netty/netty) 和 [libuv](https://github.com/libuv/libuv)，这也使得 `gnet` 达到了一个远超 Go [net](https://golang.org/pkg/net/) 的性能表现。

`gnet` 设计开发的初衷不是为了取代 Go 的标准网络库：[net](https://golang.org/pkg/net/)，而是为了创造出一个类似于 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 能高效处理网络包的 Go 语言网络服务器框架。

`gnet` 的卖点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix Domain Socket）网络框架，开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但拥有更丰富的功能特性，且性能远胜之。**

## 🚀 功能

- [x] [高性能](#-性能测试) 的基于多线程/Go程网络模型的 event-loop 事件驱动
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 内置 bytes 内存池，由开源库 [bytebufferpool](https://github.com/valyala/bytebufferpool) 提供支持
- [x] 整个生命周期是无锁的
- [x] 简洁的 APIs
- [x] 基于 Ring-Buffer 的高效内存利用
- [x] 支持多种网络协议/IPC 机制：`TCP`、`UDP` 和 `Unix Domain Socket`
- [x] 支持多种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`
- [x] 支持两种事件驱动机制：**Linux** 里的 `epoll` 以及 **FreeBSD/DragonFly/Darwin** 里的 `kqueue`
- [x] 支持异步写操作
- [x] 灵活的事件定时器
- [x] SO_REUSEPORT 端口重用
- [x] 内置多种编解码器，支持对 TCP 数据流分包：LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec 和 LengthFieldBasedFrameCodec，参考自 [netty codec](https://netty.io/4.1/api/io/netty/handler/codec/package-summary.html)，而且支持自定制编解码器
- [x] 支持 Windows 平台，基于 ~~IOCP 事件驱动机制~~ Go 标准网络库
- [ ] 实现 `gnet` 客户端

## 📚 文档

更多关于如何使用 `gnet` 的的细节，请前往 <a href="https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc" target="_blank">gnet 文档</a>。