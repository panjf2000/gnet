---
id: overview-cn
last_modified_on: "2024-03-04"
title: "预览"
description: "宏观角度陈述 gnet 框架。"
---

## gnet 是什么?

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。这个框架是基于 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 从零开发的，而且相比 Go [net](https://golang.org/pkg/net/)，它能以更低的内存占用实现更高的性能。

`gnet` 和 [net](https://golang.org/pkg/net/) 有着不一样的网络编程模式。因此，用 `gnet` 开发网络应用和用 [net](https://golang.org/pkg/net/) 开发区别很大，而且两者之间不可调和。社区里有其他同类的产品像是 [libevent](https://github.com/libevent/libevent), [libuv](https://github.com/libuv/libuv), [netty](https://github.com/netty/netty), [twisted](https://github.com/twisted/twisted), [tornado](https://github.com/tornadoweb/tornado)，`gnet` 的底层工作原理和这些框架非常类似。

`gnet` 不是为了取代 [net](https://golang.org/pkg/net/) 而生的，而是在 Go 生态中为开发者提供一个开发性能敏感的网络服务的替代品。也正因如此，`gnet` 在功能上的全面性并不如 Go [net](https://golang.org/pkg/net/)，它只会提供网络应用所需的最核心的功能和最精简的 APIs，而且 `gnet` 也并没有打算变成一个无所不包的网络框架，因为我觉得 Go [net](https://golang.org/pkg/net/) 在这方面已经做得足够好了。

`gnet` 的卖点在于它是一个高性能、轻量级、非阻塞的纯 Go 语言实现的传输层（TCP/UDP/Unix Domain Socket）网络框架。开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络服务。比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但拥有更丰富的功能特性，且性能远胜之。**

## 功能

- [x] 基于多线程/协程网络模型的[高性能](#-性能测试)事件驱动循环
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 整个生命周期是无锁的
- [x] 简单易用的 APIs
- [x] 高效、可重用而且自动伸缩的内存 buffer：(Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] 多种网络协议/IPC 机制：`TCP`、`UDP` 和 `Unix Domain Socket`
- [x] 多种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`
- [x] 两种事件驱动机制：**Linux** 里的 `epoll` 以及 **FreeBSD/DragonFly/Darwin** 里的 `kqueue`
- [x] 灵活的事件定时器
- [x] 实现 `gnet` 客户端
- [x] 支持 **Windows** 平台 (仅用于开发环境的兼容性，不要在生产环境中使用)
- [ ] 多网络地址绑定
- [ ] 支持 **TLS**
- [ ] 支持 [io_uring](https://kernel.dk/io_uring.pdf)

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

你可能会问一个问题：如果我的业务逻辑是阻塞的，那么在 `EventHandler.OnTraffic` 注册方法里的逻辑也会阻塞，从而导致阻塞 event-loop 线程，这时候怎么办？

正如你所知，基于 `gnet` 编写你的网络服务器有一条最重要的原则：永远不能让你业务逻辑（一般写在 `EventHandler.OnTraffic` 里）阻塞 event-loop 线程，这也是 `netty` 的一条最重要的原则，否则的话将会极大地降低服务器的吞吐量。

我的回答是，基于`gnet` 的另一种多线程/Go程网络模型：『带线程/Go程池的主从多 Reactors』可以解决阻塞问题，这个新网络模型通过引入一个 worker pool 来解决业务逻辑阻塞的问题：它会在启动的时候初始化一个 worker pool，然后在把 `EventHandler.OnTraffic` 里面的阻塞代码放到 worker pool 里执行，从而避免阻塞 event-loop 线程。

模型的架构图如下所示：

<p align="center">
<img alt="multi_reactor_thread_pool" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool.png" />
</p>

它的运行流程如下面的时序图：
<p align="center">
<img alt="multi-reactors" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool-sequence-diagram.png" />
</p>

`gnet` 通过利用 [ants](https://github.com/panjf2000/ants) goroutine 池（一个基于 Go 开发的高性能的 goroutine 池 ，实现了对大规模 goroutines 的调度管理、goroutines 复用）来实现『主从多 Reactors + 线程/Go程池』网络模型。关于 `ants` 的全部功能和使用，可以在 [ants 文档](https://pkg.go.dev/github.com/panjf2000/ants/v2?tab=doc) 里找到。

`gnet` 内部集成了 `ants` 以及提供了 `pool.goroutine.Default()` 方法来初始化一个 `ants` goroutine 池，然后你可以把 `EventHandler.OnTraffic` 中阻塞的业务逻辑提交到 goroutine 池里执行，最后在 goroutine 池里的代码调用 `gnet.Conn.AsyncWrite([]byte)` 方法把处理完阻塞逻辑之后得到的输出数据异步写回客户端，这样就可以避免阻塞 event-loop 线程。

有关在 `gnet` 里使用 `ants` goroutine 池的细节可以到[这里](#带阻塞逻辑的-echo-服务器)进一步了解。

## 关键设计

###  弹性内存 Buffer

#### Elastic Ring Buffer
<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif" />
</p>

#### Elastic Ring&Linked-list Buffer

<p align="center">
<img src="https://res.strikefreedom.top/static_res/blog/figures/elastic-buffer.png" />
</p>

`gnet` 内置了inbound 和 outbound 两个 buffers，分别用来缓冲输入输出的网络数据以及管理内存，gnet 里面的 inbound 和 outbound buffer 经过设计和调优，达到重用内存以及按需扩缩容的目的。

对于 TCP 协议的流数据，使用 `gnet` 不需要业务方为了解析应用层协议而自己维护和管理 buffers，`gnet` 会替业务方完成缓冲和管理网络数据的任务，降低业务代码的复杂性以及降低开发者的心智负担，使得开发者能够专注于业务逻辑而非一些底层实现。

