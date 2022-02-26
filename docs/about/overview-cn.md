---
id: overview-cn
last_modified_on: "2020-07-10"
title: "预览"
description: "宏观角度陈述 gnet 框架。"
---

## gnet 是什么?

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。它直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Go 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[netty](https://github.com/netty/netty) 和 [libuv](https://github.com/libuv/libuv)，这也使得 `gnet` 达到了一个远超 Go [net](https://golang.org/pkg/net/) 的性能表现。

`gnet` 设计开发的初衷不是为了取代 Go 的标准网络库：[net](https://golang.org/pkg/net/)，而是为了创造出一个类似于 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 能高效处理网络包的 Go 语言网络服务器框架。

`gnet` 的卖点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix Domain Socket）网络框架，开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但拥有更丰富的功能特性，且性能远胜之。**

## 功能

- [x] [高性能](#-性能测试) 的基于多线程/Go程网络模型的 event-loop 事件驱动
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 整个生命周期是无锁的
- [x] 简单易用的 APIs
- [x] 高效、可重用而且自动伸缩的弹性内存 buffers：Ring-Buffer, Linked-List-Buffer 以及结合了前两者的 Mixed-Buffer
- [x] 支持多种网络协议/IPC 机制：`TCP`、`UDP` 和 `Unix Domain Socket`
- [x] 支持多种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`
- [x] 支持两种事件驱动机制：**Linux** 里的 `epoll` 以及 **FreeBSD/DragonFly/Darwin** 里的 `kqueue`
- [x] 灵活的事件定时器
- [x] 实现 `gnet` 客户端

## 架构
### 多线程/Go程网络模型
#### 主从多 Reactors

`gnet` 重新设计开发了一个新内置的多线程/Go程网络模型：『主从多 Reactors』，这也是 `netty` 默认的多线程网络模型，下面是这个模型的原理图：

<p align="center">
<img alt="multi_reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors.png" />
</p>

它的运行流程如下面的时序图：
<p align="center">
<img alt="reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors-sequence-diagram.png" />
</p>

#### 主从多 Reactors + 线程/Go程池

你可能会问一个问题：如果我的业务逻辑是阻塞的，那么在 `EventHandler.React` 注册方法里的逻辑也会阻塞，从而导致阻塞 event-loop 线程，这时候怎么办？

正如你所知，基于 `gnet` 编写你的网络服务器有一条最重要的原则：永远不能让你业务逻辑（一般写在 `EventHandler.React` 里）阻塞 event-loop 线程，这也是 `netty` 的一条最重要的原则，否则的话将会极大地降低服务器的吞吐量。

我的回答是，基于`gnet` 的另一种多线程/Go程网络模型：『带线程/Go程池的主从多 Reactors』可以解决阻塞问题，这个新网络模型通过引入一个 worker pool 来解决业务逻辑阻塞的问题：它会在启动的时候初始化一个 worker pool，然后在把 `EventHandler.React`里面的阻塞代码放到 worker pool 里执行，从而避免阻塞 event-loop 线程。

模型的架构图如下所示：

<p align="center">
<img alt="multi_reactor_thread_pool" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool.png" />
</p>

它的运行流程如下面的时序图：
<p align="center">
<img alt="multi-reactors" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool-sequence-diagram.png" />
</p>

`gnet` 通过利用 [ants](https://github.com/panjf2000/ants) goroutine 池（一个基于 Go 开发的高性能的 goroutine 池 ，实现了对大规模 goroutines 的调度管理、goroutines 复用）来实现『主从多 Reactors + 线程/Go程池』网络模型。关于 `ants` 的全部功能和使用，可以在 [ants 文档](https://pkg.go.dev/github.com/panjf2000/ants/v2?tab=doc) 里找到。

`gnet` 内部集成了 `ants` 以及提供了 `pool.goroutine.Default()` 方法来初始化一个 `ants` goroutine 池，然后你可以把 `EventHandler.React` 中阻塞的业务逻辑提交到 goroutine 池里执行，最后在 goroutine 池里的代码调用 `gnet.Conn.AsyncWrite([]byte)` 方法把处理完阻塞逻辑之后得到的输出数据异步写回客户端，这样就可以避免阻塞 event-loop 线程。

有关在 `gnet` 里使用 `ants` goroutine 池的细节可以到[这里](#带阻塞逻辑的-echo-服务器)进一步了解。

## 关键设计

###  弹性内存 Buffer

#### Elastic Ring Buffer
<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif" />
</p>

#### Elastic Ring&Linked-list Buffer

<p align="center">
<img src="https://img.taohuawu.club/gallery/elastic-buffer.png" />
</p>

`gnet` 内置了inbound 和 outbound 两个 buffers，分别用来缓冲输入输出的网络数据以及管理内存，gnet 里面的 inbound 和 outbound buffer 经过设计和调优，达到重用内存以及按需扩缩容的目的。

对于 TCP 协议的流数据，使用 `gnet` 不需要业务方为了解析应用层协议而自己维护和管理 buffers，`gnet` 会替业务方完成缓冲和管理网络数据的任务，降低业务代码的复杂性以及降低开发者的心智负担，使得开发者能够专注于业务逻辑而非一些底层实现。

