---
id: doc-for-gnet-v1-zh
last_modified_on: "2020-03-14"
title: "gnet v1 文档"
description: "最快的 Go 网络框架 gnet 来啦！"
author_github: https://github.com/panjf2000
tags: ["type: 官宣", "domain: 展示"]
---

# 🎉 开始使用

## 前提

`gnet` 需要 Go 版本 >= 1.9。

## 安装

```powershell
go get -u github.com/panjf2000/gnet
```

`gnet` 支持作为一个 Go module 被导入，基于 [Go 1.11 Modules](https://github.com/golang/go/wiki/Modules) (Go 1.11+)，只需要在你的项目里直接 `import "github.com/panjf2000/gnet"`，然后运行 `go [build|run|test]` 自动下载和构建需要的依赖包。

## 使用示例

**详细的文档在这里: [gnet 接口文档](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc)，不过下面我们先来了解下使用 `gnet` 的简略方法。**

用 `gnet` 来构建网络服务器是非常简单的，只需要实现 `gnet.EventHandler`接口然后把你关心的事件函数注册到里面，最后把它连同监听地址一起传递给 `gnet.Serve` 函数就完成了。在服务器开始工作之后，每一条到来的网络连接会在各个事件之间传递，如果你想在某个事件中关闭某条连接或者关掉整个服务器的话，直接在事件函数里把 `gnet.Action` 设置成 `Close` 或者 `Shutdown` 就行了。

Echo 服务器是一种最简单网络服务器，把它作为 `gnet` 的入门例子在再合适不过了，下面是一个最简单的 echo server，它监听了 9000 端口：
### 不带阻塞逻辑的 echo 服务器

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

正如你所见，上面的例子里 `gnet` 实例只注册了一个 `EventHandler.React` 事件。一般来说，主要的业务逻辑代码会写在这个事件方法里，这个方法会在服务器接收到客户端写过来的数据之时被调用，此时的输入参数: `frame` 已经是解码过后的一个完整的网络数据包，一般来说你需要实现 `gnet` 的 [codec 接口](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#ICodec)作为你自己的业务编解码器来处理 TCP 组包和分包的问题，如果你不实现那个接口的话，那么 `gnet` 将会使用[默认的 codec](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#BuiltInFrameCodec)，这意味着在 `EventHandler.React` 被触发调用之时输入参数: `frame` 里存储的是所有网络数据：包括最新的以及还在 buffer 里的旧数据，然后处理输入数据（这里只是把数据 echo 回去）并且在处理完之后把需要输出的数据赋值给 `out` 变量并返回，接着输出的数据会经过编码，最后被写回客户端。

### 带阻塞逻辑的 echo 服务器

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

正如我在『主从多 Reactors + 线程/Go程池』那一节所说的那样，如果你的业务逻辑里包含阻塞代码，那么你应该把这些阻塞代码变成非阻塞的，比如通过把这部分代码放到独立的 goroutines 去运行，但是要注意一点，如果你的服务器处理的流量足够的大，那么这种做法将会导致创建大量的 goroutines 极大地消耗系统资源，所以我一般建议你用 goroutine pool 来做 goroutines 的复用和管理，以及节省系统资源。

**各种 gnet 示例:**

- [v1.6.0 版本之前的示例](https://github.com/gnet-io/gnet-examples/tree/984eed8bda54282f0d6cb2005ff48e529ae8980d)

- [v1.6.x 系列的示例](https://github.com/gnet-io/gnet-examples/tree/master)

- [v2.x.x 系列的示例](https://github.com/gnet-io/gnet-examples)

## I/O 事件

 `gnet` 目前支持的 I/O 事件如下：

- `EventHandler.OnInitComplete` 当 server 初始化完成之后调用。
- `EventHandler.OnOpened` 当连接被打开的时候调用。
- `EventHandler.OnClosed` 当连接被关闭的之后调用。
- `EventHandler.React` 当 server 端接收到从 client 端发送来的数据的时候调用。（你的核心业务代码一般是写在这个方法里）
- `EventHandler.Tick` 服务器启动的时候会调用一次，之后就以给定的时间间隔定时调用一次，是一个定时器方法。
- `EventHandler.PreWrite` 预先写数据方法，在 server 端写数据回 client 端之前调用。

## poll_opt 模式

默认情况下，`gnet` 使用官方包 `golang.org/x/sys/unix` 实现基于 `epoll` 和 `kqueue` 的网络轮询器 poller，这种实现需要引入一个 `fd->conn` 哈希表，通过它可以用文件描述符 `fd` 找到对应的 `connection` 结构体，但现在用户可以在 `go build` 编译项目的时候加入构建标签 `poll_opt`，像这样：`go build -tags=poll_opt`，然后 `gnet` 会切换到另一种优化的实现，直接调用 `epoll` 或 `kqueue` 的系统调用，将文件描述符添加到监控列表中，同时将相应的 connection 结构体指针存储到 `epoll_data` 或 `kevent` 中，在这种情况下，`gnet` 就能在 I/O 事件循环中摆脱掉 `fd->conn` 哈希表，将 `void*` 指针转换成 `connection` 结构体指针，通过这种优化，理论上应该可以得到更高的性能。

代码细节请浏览 [#230](https://github.com/panjf2000/gnet/pull/230)。

## 定时器

`EventHandler.Tick` 会每隔一段时间触发一次，间隔时间你可以自己控制，设定返回的 `delay` 变量就行。

定时器的第一次触发是在 gnet server 启动之后，如果你要设置定时器，别忘了设置 option 选项：`WithTicker(true)`。

```go
events.Tick = func() (delay time.Duration, action Action){
	log.Printf("tick")
	delay = time.Second
	return
}
```

## UDP 支持

`gnet` 支持 UDP 协议，所以在 `gnet.Serve` 里绑定允许绑定 UDP 地址，`gnet` 的 UDP 支持有如下的特性：

- 网络数据的读入和写出不做缓冲，会一次性读写客户端，也就是说 `gnet.Conn` 所有那些操作内部的 buffer 的函数都不可用，比如 `c.Read()`, `c.ResetBuffer()`, `c.BufferLength()` 和其他 buffer 相关的函数；使用者不能调用上述那些函数去操作数据，而应该直接使用 `gnet.React(frame []byte, c gnet.Conn)` 函数入参中的 `frame []byte` 作为 UDP 数据包。
- `EventHandler.OnOpened` 和 `EventHandler.OnClosed` 这两个事件在 UDP 下不可用，唯一可用的事件是 `React`。
- TCP 里的异步写操作是 `AsyncWrite([]byte)` 方法，而在 UDP 里对应的方法是  `SendTo([]byte)`。

## Unix Domain Socket 支持

`gnet` 还支持 UDS(Unix Domain Socket) 机制，只需要把类似 "unix://xxx" 的 UDS 地址传参给 `gnet.Serve` 函数绑定就行了。

在 `gnet` 里使用 UDS 和使用 TCP 没有什么不同，所有 TCP 协议下可以使用的事件函数都可以在 UDS 中使用。

## 使用多核

`gnet.WithMulticore(true)` 参数指定了 `gnet` 是否会使用多核来进行服务，如果是 `true` 的话就会使用多核，否则就是单核运行，利用的核心数一般是机器的 CPU 数量。

## 负载均衡

`gnet` 目前支持三种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`，你可以通过传递 functional option 的 `LB` (RoundRobin/LeastConnections/SourceAddrHash) 的值给 `gnet.Serve` 来指定要使用的负载均衡算法。

如果没有显示地指定，那么 `gnet` 将会使用 `Round-Robin` 作为默认的负载均衡算法。

## SO_REUSEPORT 端口复用

服务器支持 [SO_REUSEPORT](https://lwn.net/Articles/542629/) 端口复用特性，允许多个 sockets 监听同一个端口，然后内核会帮你做好负载均衡，每次只唤醒一个 socket 来处理 `connect` 请求，避免惊群效应。

默认情况下，`gnet` 不会有惊群效应，因为 `gnet` 默认的网络模型是主从多 Reactors，只会有一个主 reactor 在监听端口以及接收新连接。所以，开不开启 `SO_REUSEPORT` 选项是无关紧要的，只是开启了这个选项之后 `gnet` 的网络模型将会切换成 `evio` 的旧网络模型，这一点需要注意一下。

开启这个功能也很简单，使用 functional options 设置一下即可：

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true), gnet.WithReusePort(true)))
```

## 多种内置的 TCP 流编解码器

`gnet` 内置了多种用于 TCP 流分包的编解码器。

目前一共实现了 4 种常见的编解码器：LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec 和 LengthFieldBasedFrameCodec，基本上能满足大多数应用场景的需求了；而且 `gnet` 还允许用户实现自己的编解码器：只需要实现 [gnet.ICodec](https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc#ICodec) 接口，并通过 functional options 替换掉内部默认的编解码器即可。

这里有一个使用编解码器对 TCP 流分包的[例子](https://github.com/gnet-io/gnet-examples/tree/master/examples/codec)。

# 📊 性能测试

## TechEmpower 性能测试

```bash
# 硬件环境
* 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 3.20GHz
* 32GB RAM
* Dedicated Cisco 10-gigabit Ethernet switch
* Debian 12 "bookworm"
* Go1.19.x linux/amd64
```

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-top50-light.jpg)

这是包含全部编程语言框架的性能排名***前 50*** 的结果，总榜单包含了全世界共计 ***486*** 个框架，其中 `gnet` 排名***第一***。

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-topN-go-light.png)

这是 Go 语言分类下的全部排名，`gnet` 超越了其他所有框架，位列第一，是***最快***的 Go 网络框架。

完整的排行可以通过 [TechEmpower Benchmark **Round 22**](https://www.techempower.com/benchmarks/#hw=ph&test=plaintext&section=data-r22) 查看。

## 同类型的网络库性能对比

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

# ️🚨 证书

`gnet` 的源码允许用户在遵循 [MIT 开源证书](https://github.com/panjf2000/gnet/blob/master/LICENSE) 规则的前提下使用。

# 👏 贡献者

请在提 PR 之前仔细阅读 [Contributing Guidelines](https://github.com/panjf2000/gnet/blob/master/CONTRIBUTING.md)，感谢那些为 `gnet` 贡献过代码的开发者！

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# 🙏 致谢

- [evio](https://github.com/tidwall/evio)
- [netty](https://github.com/netty/netty)
- [ants](https://github.com/panjf2000/ants)
- [go_reuseport](https://github.com/kavu/go_reuseport)
- [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [goframe](https://github.com/smallnest/goframe)
- [ringbuffer](https://github.com/smallnest/ringbuffer)

# ⚓ 相关文章

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoller 原生网络模型之源码全面揭秘](https://strikefreedom.top/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://strikefreedom.top/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://strikefreedom.top/releasing-gnet-v1-with-techempower)

# 🎡 用户案例

以下公司/组织在生产环境上使用了 `gnet` 作为底层网络服务。

<a href="https://www.tencent.com"><img src="https://res.strikefreedom.top/static_res/logos/tencent_logo.png" width="250" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.iqiyi.com" target="_blank"><img src="https://res.strikefreedom.top/static_res/logos/iqiyi-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.mi.com" target="_blank"><img src="https://res.strikefreedom.top/static_res/logos/mi-logo.png" width="150" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.360.com" target="_blank"><img src="https://res.strikefreedom.top/static_res/logos/360-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://tieba.baidu.com/" target="_blank"><img src="https://res.strikefreedom.top/static_res/logos/baidu-tieba-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://game.qq.com/" target="_blank"><img src="https://res.strikefreedom.top/static_res/logos/tencent-games-logo.png" width="200" align="middle"/></a>

如果你的项目也在使用 `gnet`，欢迎给我提 [Pull Request](https://github.com/panjf2000/gnet/pulls) 来更新这份用户案例列表。

# 💰 支持

如果有意向，可以通过每个月定量的少许捐赠来支持这个项目。

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg" /></a>

# 💎 赞助

每月定量捐赠 10 刀即可成为本项目的赞助者，届时您的 logo 或者 link 可以展示在本项目的 README 上。

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg" /></a>

# ☕️ 打赏

> 当您通过以下方式进行捐赠时，请务必留下姓名、Github账号或其他社交媒体账号，以便我将其添加到捐赠者名单中，以表谢意。

<table><tr>
<td><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" /></td>
<td><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" /></td>
<td><a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" /></a></td>
</tr></table>

# 💴 资助者

<a target="_blank" href="https://github.com/patrick-othmer"><img src="https://avatars1.githubusercontent.com/u/8964313" width="100" alt="Patrick Othmer" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://avatars2.githubusercontent.com/u/50285334" width="100" alt="Jimmy" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/cafra"><img src="https://avatars0.githubusercontent.com/u/13758306" width="100" alt="ChenZhen" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/yangwenmai"><img src="https://avatars0.githubusercontent.com/u/1710912" width="100" alt="Mai Yang" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/BeijingWks"><img src="https://avatars3.githubusercontent.com/u/33656339" width="100" alt="王开帅" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/refs"><img src="https://avatars3.githubusercontent.com/u/6905948" width="100" alt="Unger Alejandro" /></a>&nbsp;&nbsp;<a target="_blank" href="https://github.com/Swaggadan"><img src="https://avatars.githubusercontent.com/u/137142" width="100" alt="Swaggadan" /></a>&nbsp;<a target="_blank" href="https://github.com/Wuvist"><img src="https://avatars.githubusercontent.com/u/657796" width="100" alt="Weng Wei" /></a>

# 🔑 JetBrains 开源证书支持

`gnet` 项目一直以来都是在 JetBrains 公司旗下的 GoLand 集成开发环境中进行开发，基于 **free JetBrains Open Source license(s)** 正版免费授权，在此表达我的谢意。
<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle" /></a>

# 🔋 赞助商

<p>
	<h3>本项目由以下机构赞助：</h3>
	<a href="https://www.digitalocean.com/"><img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/SVG/DO_Logo_horizontal_blue.svg" width="201px" />
	</a>
</p>