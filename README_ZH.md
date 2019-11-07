<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" width="300" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<a title="gnet on Sourcegraph" target="_blank" href="https://sourcegraph.com/github.com/panjf2000/gnet?badge"><img src="https://sourcegraph.com/github.com/panjf2000/gnet/-/badge.svg?style=flat-square"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Doc for gnet" target="_blank" href="https://gowalker.org/github.com/panjf2000/gnet?lang=zh-CN"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
</p>

[英文](README.md) | 🇨🇳中文

# 📖 简介

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。它直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Golang 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[netty](https://github.com/netty/netty) 和 [libuv](https://github.com/libuv/libuv)。

这个项目存在的价值是提供一个在网络包处理方面能和 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 这两个项目具有相近性能的 Go 语言网络服务器框架。

`gnet` 的亮点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix-Socket）网络框架，开发者可以使用 `gnet` 来实现自己的应用层网络协议，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但性能远胜之。**

# 🚀 功能

- [x] [高性能](#-性能测试) 的基于多线程/Go程模型的 event-loop 事件驱动
- [x] 内置 Round-Robin 轮询负载均衡算法
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 内置 bytes 内存池，由开源库 [pool](https://github.com/gobwas/pool/) 提供支持
- [x] 简洁的 APIs
- [x] 基于 Ring-Buffer 的高效内存利用
- [x] 支持多种网络协议：TCP、UDP、Unix Sockets
- [x] 支持两种事件驱动机制：Linux 里的 epoll 以及 FreeBSD 里的 kqueue
- [x] 支持异步写操作
- [x] 灵活的事件定时器
- [x] SO_REUSEPORT 端口重用
- [x] 新增多种编解码器，支持 TCP 数据流分包：LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec 和 LengthFieldBasedFrameCodec，参考自 [netty codec](https://github.com/netty/netty/tree/4.1/codec/src/main/java/io/netty/handler/codec)
- [ ] 加入更多的负载均衡算法：随机、最少连接、一致性哈希等等
- [ ] 支持 Windows 平台的 IOCP 事件驱动机制
- [ ] 支持 TLS
- [ ] 实现 `gnet` 客户端

# 💡 核心设计
## 多线程/Go程模型
### 主从多 Reactors 模型

`gnet` 重新设计开发了一个新内置的多线程/Go程模型：『主从多 Reactors』，这也是 `netty` 默认的线程模型，下面是这个模型的原理图：

<p align="center">
<img width="820" alt="multi_reactor" src="https://user-images.githubusercontent.com/7496278/64916634-8f038080-d7b3-11e9-82c8-f77e9791df86.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img width="869" alt="reactor" src="https://user-images.githubusercontent.com/7496278/64918644-a5213900-d7d3-11e9-88d6-1ec1ec72c1cd.png">
</p>

### 主从多 Reactors + 线程/Go程池

你可能会问一个问题：如果我的业务逻辑是阻塞的，那么在 `EventHandler.React` 注册方法里的逻辑也会阻塞，从而导致阻塞 event-loop 线程，这时候怎么办？

正如你所知，基于 `gnet` 编写你的网络服务器有一条最重要的原则：永远不能让你业务逻辑（一般写在 `EventHandler.React` 里）阻塞 event-loop 线程，否则的话将会极大地降低服务器的吞吐量，这也是 `netty` 的一条最重要的原则。

我的回答是，基于`gnet` 的另一种多线程/Go程模型：『带线程/Go程池的主从多 Reactors』可以解决阻塞问题，这个新网络模型通过引入一个 worker pool 来解决业务逻辑阻塞的问题：它会在启动的时候初始化一个 worker pool，然后在把 `EventHandler.React`里面的阻塞代码放到 worker pool 里执行，从而避免阻塞 event-loop 线程，

模型的架构图如下所示：

<p align="center">
<img width="854" alt="multi_reactor_thread_pool" src="https://user-images.githubusercontent.com/7496278/64918783-90de3b80-d7d5-11e9-9190-ff8277c95db1.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img width="916" alt="multi-reactors" src="https://user-images.githubusercontent.com/7496278/64918646-a7839300-d7d3-11e9-804a-d021ddd23ca3.png">
</p>

`gnet` 通过利用 [ants](https://github.com/panjf2000/ants) goroutine 池（一个基于 Go 开发的高性能的 goroutine 池 ，实现了对大规模 goroutines 的调度管理、goroutines 复用）来实现『主从多 Reactors + 线程/Go程池』网络模型。关于 `ants` 的全部功能和使用，可以在 [ants 文档](https://gowalker.org/github.com/panjf2000/ants?lang=zh-CN) 里找到。

`gnet` 内部集成了 `ants` 以及提供了 `pool.NewWorkerPool` 方法来初始化一个 `ants` goroutine 池，然后你可以把 `EventHandler.React` 中阻塞的业务逻辑提交到 goroutine 池里执行，最后在 goroutine 池里的代码调用 `gnet.Conn.AsyncWrite` 方法把处理完阻塞逻辑之后得到的输出数据异步写回客户端，这样就可以避免阻塞 event-loop 线程。

有关在 `gnet` 里使用 `ants` goroutine 池的细节可以到[这里](#echo-server-with-blocking-logic)进一步了解。

## 自动扩容的 Ring-Buffer

`gnet` 利用 Ring-Buffer 来缓冲网络数据以及管理内存。

<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916810-4f8b6300-d7b8-11e9-9459-5517760da738.gif">
</p>


# 🎉 开始使用

## 前提

`gnet` 需要 Go 版本 >= 1.9。

## 安装

```powershell
go get -u github.com/panjf2000/gnet
```

`gnet` 支持作为一个 Go module 被导入，基于 [Go 1.11 Modules](https://github.com/golang/go/wiki/Modules) (Go 1.11+)，只需要在你的项目里直接 `import "github.com/panjf2000/gnet"`，然后运行 `go [build|run|test]` 自动下载和构建需要的依赖包。

## 使用示例

**详细的文档在这里: [gnet 接口文档](https://gowalker.org/github.com/panjf2000/gnet?lang=zh-CN)，不过下面我们先来了解下使用 `gnet` 的简略方法。**

用 `gnet` 来构建网络服务器是非常简单的，只需要实现 `gnet.EventHandler`接口然后把你关心的事件函数注册到里面，最后把它连同监听地址一起传递给 `gnet.Serve` 函数就完成了。在服务器开始工作之后，每一条到来的网络连接会在各个事件之间传递，如果你想在某个事件中关闭某条连接或者关掉整个服务器的话，直接把 `gnet.Action` 设置成 `Cosed` 或者 `Shutdown`就行了。

Echo 服务器是一种最简单网络服务器，把它作为 `gnet` 的入门例子在再合适不过了，下面是一个最简单的 echo server，它监听了 9000 端口：
### 不带阻塞逻辑的 echo 服务器
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
	out = c.Read()
	c.ResetBuffer()
	return
}

func main() {
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```

正如你所见，上面的例子里 `gnet` 实例只注册了一个 `EventHandler.React` 事件。一般来说，主要的业务逻辑代码会写在这个事件方法里，这个方法会在服务器接收到客户端写过来的数据之时被调用，然后处理输入数据（这里只是把数据 echo 回去）并且在处理完之后把需要输出的数据赋值给 `out` 变量然后返回，之后你就不用管了，`gnet` 会帮你把数据写回客户端的。

### 带阻塞逻辑的 echo 服务器
```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool"
)

type echoServer struct {
	*gnet.EventServer
	pool *pool.WorkerPool
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
	p := pool.NewWorkerPool()
	defer p.Release()
	
	echo := &echoServer{pool: p}
	log.Fatal(gnet.Serve(echo, "tcp://:9000", gnet.WithMulticore(true)))
}
```
正如我在『主从多 Reactors + 线程/Go程池』那一节所说的那样，如果你的业务逻辑里包含阻塞代码，那么你应该把这些阻塞代码变成非阻塞的，比如通过把这部分代码通过 goroutine 去运行，但是要注意一点，如果你的服务器处理的流量足够的大，那么这种做法将会导致创建大量的 goroutines 极大地消耗系统资源，所以我一般建议你用 goroutine pool 来做 goroutines 的复用和管理，以及节省系统资源。

**更多的例子可以在这里查看: [gnet 示例](https://github.com/panjf2000/gnet/tree/master/examples)。**

## I/O 事件

 `gnet` 目前支持的 I/O 事件如下：

- `EventHandler.OnInitComplete` 当 server 初始化完成之后调用。
- `EventHandler.OnOpened` 当连接被打开的时候调用。
- `EventHandler.OnClosed` 当连接被关闭的时候调用。
- `EventHandler.React` 当 server 端接收到从 client 端发送来的数据的时候调用。（你的核心业务代码一般是写在这个方法里）
- `EventHandler.Tick` 服务器启动的时候会调用一次，之后就以给定的时间间隔定时调用一次，是一个定时器方法。
- `EventHandler.PreWrite` 预先写数据方法，在 server 端写数据回 client 端之前调用。


## 定时器

`EventHandler.Tick` 会每隔一段时间触发一次，间隔时间你可以自己控制，设定返回的 `delay` 变量就行。

定时器的第一次触发是在 `gnet.Serving` 事件之后，如果你要设置定时器，别忘了设置 option 选项：`WithTicker(true)`。

```go
events.Tick = func() (delay time.Duration, action Action){
	log.Printf("tick")
	delay = time.Second
	return
}
```

## UDP 支持

`gnet` 支持 UDP 协议，在 `gnet.Serve` 里绑定 UDP 地址即可，`gnet` 的 UDP 支持有如下的特性：

- 数据进入服务器之后立刻写回客户端，不做缓存。
-  `EventHandler.OnOpened` 和 `EventHandler.OnClosed` 这两个事件在 UDP 下不可用，唯一可用的事件是 `React`。

## 使用多核

`gnet.WithMulticore(true)` 参数指定了 `gnet` 是否会使用多核来进行服务，如果是 `true` 的话就会使用多核，否则就是单核运行，利用的核心数一般是机器的 CPU 数量。

## 负载均衡

`gnet` 目前内置的负载均衡算法是轮询调度 Round-Robin，暂时不支持自定制。

## SO_REUSEPORT 端口复用

服务器支持 [SO_REUSEPORT](https://lwn.net/Articles/542629/) 端口复用特性，允许多个 sockets 监听同一个端口，然后内核会帮你做好负载均衡，每次只唤醒一个 socket 来处理 accept 请求，避免惊群效应。

开启这个功能也很简单，使用 functional options 设置一下即可：

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true), gnet.WithReusePort(true)))
```

# 📊 性能测试

## 同类型的网络库性能对比

## Linux (epoll)

### 系统参数

```powershell
# Machine information
        OS : Ubuntu 18.04/x86_64
       CPU : 8 Virtual CPUs
    Memory : 16.0 GiB

# Go version and configurations
Go Version : go1.12.9 linux/amd64
GOMAXPROCS=8
```

#### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_linux.png)

#### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_linux.png)

## FreeBSD (kqueue)

### 系统参数

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

# 📄 证书

`gnet` 的源码允许用户在遵循 MIT [开源证书](/LICENSE) 规则的前提下使用。

# 👏 贡献者

请在提 PR 之前仔细阅读 [Contributing Guidelines](CONTRIBUTING.md)，感谢那些为 `gnet` 贡献过代码的开发者！

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# 🙏 致谢

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [pool](https://github.com/gobwas/pool)
- [goframe](https://github.com/smallnest/goframe)

# 📚 相关文章

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://taohuawu.club/go-event-loop-networking-library-gnet)

## JetBrains 开源证书支持

`gnet` 项目一直以来都是在 JetBrains 公司旗下的 GoLand 集成开发环境中进行开发，基于 **free JetBrains Open Source license(s)** 正版免费授权，在此表达我的谢意。

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>

# 💰 支持

如果有意向，可以通过每个月定量的少许捐赠来支持这个项目。

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg"></a>

# 💎 赞助

每月定量捐赠 10 刀即可成为本项目的赞助者，届时您的 logo 或者 link 可以展示在本项目的 README 上。

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg"></a>

# ☕️ 打赏

<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" align="middle"/></a>&nbsp;&nbsp;