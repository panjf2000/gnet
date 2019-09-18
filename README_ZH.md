<p align="center">
<img src="logo.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Godoc for gnet" target="_blank" href="https://godoc.org/github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?style=flat-square"></a>
</p>

# [[英文]](README.md)

`gnet` 是一个基于 Event-Loop 事件驱动的高性能和轻量级网络库。这个库直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Golang 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[libuv](https://github.com/libuv/libuv) 和 [libevent](https://github.com/libevent/libevent)。

这个项目存在的价值是提供一个在网络包处理方面能和 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 这两个项目具有相近性能的 Go 语言网络服务器框架。

`gnet` 的亮点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix-Socket）网络库，开发者可以使用 `gnet` 来实现自己的应用层网络协议，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但是性能更好。**

# 功能

- [高性能](#性能测试) 的基于多线程/Go程模型的 Event-Loop 事件驱动
- 内置 Round-Robin 轮询负载均衡算法
- 简洁的 APIs
- 基于 Ring-Buffer 的高效内存利用
- 支持多种网络协议：TCP、UDP、Unix Sockets
- 支持两种事件驱动机制：Linux 里的 epoll 以及 FreeBSD 里的 kqueue
- 支持异步写操作
- 允许多个网络监听地址绑定在一个 Event-Loop 上
- 灵活的事件定时器
- SO_REUSEPORT 端口重用

# 核心设计

## 多线程/Go程模型

`gnet` 重新设计开发了一个新内置的多线程/Go程模型：『主从 Reactor 多线程/Go程』，这也是 `netty` 默认的线程模型，下面是这个模型的原理图：

<p align="center">
<img width="820" alt="multi_reactor" src="https://user-images.githubusercontent.com/7496278/64916634-8f038080-d7b3-11e9-82c8-f77e9791df86.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img width="869" alt="reactor" src="https://user-images.githubusercontent.com/7496278/64918644-a5213900-d7d3-11e9-88d6-1ec1ec72c1cd.png">
</p>

现在我正在为 `gnet` 开发一个新的多线程/Go程模型：『带线程/Go程池的主从 Reactors 多线程/Go程』，并且很快就能完成，这个模型的架构图如下所示：

<p align="center">
<img width="854" alt="multi_reactor_thread_pool" src="https://user-images.githubusercontent.com/7496278/64918783-90de3b80-d7d5-11e9-9190-ff8277c95db1.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img width="916" alt="multi-reactors" src="https://user-images.githubusercontent.com/7496278/64918646-a7839300-d7d3-11e9-804a-d021ddd23ca3.png">
</p>

## 通信机制

`gnet` 的『主从 Reactors 多线程/Go程』模型是基于 Golang 里的 Goroutines的，一个 Reactor 挂载在一个 Goroutine 上，所以在 `gnet` 的这个网络模型里主 Reactor/Goroutine 与从 Reactors/Goroutines 有海量通信的需求，因此 `gnet` 里必须要有一个能在 Goroutines 之间进行高效率的通信的机制，我没有选择 Golang 里的主流方案：基于 Channel 的 CSP 模型，而是选择了性能更好、基于 Ring-Buffer 的 Disruptor 方案。

所以我最终选择了 [go-disruptor](https://github.com/smartystreets-prototypes/go-disruptor)：高性能消息分发队列 LMAX Disruptor 的 Golang 实现。

## 自动扩容的 Ring-Buffer

`gnet` 利用 Ring-Buffer 来缓存 TCP 流数据以及管理内存使用。

<p align="center">
<img src="https://user-images.githubusercontent.com/7496278/64916810-4f8b6300-d7b8-11e9-9459-5517760da738.gif">
</p>


# 开始使用

## 安装

```sh
$ go get -u github.com/panjf2000/gnet
```

## 使用示例

用 `gnet` 来构建网络服务器是非常简单的，只需要把你关心的事件注册到 `gnet.Events` 里面，然后把它和绑定的监听地址一起传递给 `gnet.Serve` 方法就完成了。在服务器开始工作之后，每一条到来的网络连接会在各个事件之间传递，如果你想在某个事件中关闭某条连接或者关掉整个服务器的话，直接把 `gnet.Action` 设置成 `Cosed` 或者 `Shutdown`就行了。

Echo 服务器是一种最简单网络服务器，把它作为 `gnet` 的入门例子在再合适不过了，下面是一个最简单的 echo server，它监听了 9000 端口：

```go
package main

import (
	"log"

    "github.com/panjf2000/gnet"
    "github.com/panjf2000/gnet/ringbuffer"

)

func main() {
	var events gnet.Events
	events.Multicore = true
	events.React = func(c gnet.Conn, inBuf *ringbuffer.RingBuffer) (out []byte, action gnet.Action) {
		top, tail := inBuf.PreReadAll()
		out = append(top, tail...)
		inBuf.Reset()
		return
	}
	log.Fatal(gnet.Serve(events, "tcp://:9000"))
}
```

正如你所见，上面的例子里 `gnet` 实例只注册了一个 `React` 事件。一般来说，主要的业务逻辑代码会写在这个事件方法里，这个方法会在服务器接收到客户端写过来的数据之时被调用，然后处理输入数据（这里只是把数据 echo 回去）并且在处理完之后把需要输出的数据赋值给 `out` 变量然后返回，之后你就不用管了，`gnet` 会帮你把数据写回客户端的。

### I/O 事件

 `gnet` 目前支持的 I/O 事件如下：

- `OnInitComplete` 当 server 初始化完成之后调用。
- `OnOpened` 当连接被打开的时候调用。
- `OnClosed` 当连接被关闭的时候调用。
- `OnDetached` 当主动摘除连接的时候的调用。
- `React` 当 server 端接收到从 client 端发送来的数据的时候调用。（你的核心业务代码一般是写在这个方法里）
- `Tick` 服务器启动的时候会调用一次，之后就以给定的时间间隔定时调用一次，是一个定时器方法。
- `PreWrite` 预先写数据方法，在 server 端写数据回 client 端之前调用。

### 多地址绑定

```go
// 在同一个 Server 上同时绑定 TCP 和 Unix-Socket 两个地址
gnet.Serve(events, "tcp://:9000", "unix://socket")
```


### 定时器

`Tick` 会每隔一段时间触发一次，间隔时间你可以自己控制，设定返回的 `delay` 变量就行。

定时器的第一次触发是在 `gnet.Serving` 事件之后。

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
-  `OnOpened` 和 `OnClosed` 这两个事件在 UDP 下不可用，唯一可用的事件是 `React`。

## 使用多核

`Events.Multicore` 参数指定了 `gnet` 是否会使用多核来进行服务，如果是 `true` 的话就会使用多核，否则就是单核运行，利用的核心数一般是机器的 CPU 数量。

## 负载均衡

`gnet` 目前内置的负载均衡算法是轮询调度 Round-Robin，暂时不支持自定制。

## SO_REUSEPORT 端口复用

服务器支持 [SO_REUSEPORT](https://lwn.net/Articles/542629/) 端口复用特性，允许多个 sockets 监听同一个端口，然后内核会帮你做好负载均衡，每次只唤醒一个 socket 来处理 accept 请求，避免惊群效应。

开启这个功能也很简单，在要绑定的监听地址后面设置 `reuseport=true` 即可：

```go
gnet.Serve(events, "tcp://:9000?reuseport=true"))
```

# 性能测试

## Linux (epoll)

### 系统参数

```powershell
Go Version : go1.12.9 linux/amd64
        OS : Ubuntu 18.04/x86_64
       CPU : 8 Virtual CPUs
    Memory : 16.0 GiB
```

### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_linux.png)

### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_linux.png)

## FreeBSD (kqueue)

### 系统参数

```powershell
Go Version : go version go1.12.9 darwin/amd64
        OS : macOS Mojave 10.14.6/x86_64
       CPU : 4 CPUs
    Memory : 8.0 GiB
```

### Echo Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_mac.png)

### HTTP Server

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/http_mac.png)

# 证书

`gnet` 的源码允许用户在遵循 MIT [开源证书](/LICENSE) 规则的前提下使用。

# 鸣谢

- [evio](https://github.com/tidwall/evio)
- [go-disruptor](https://github.com/smartystreets-prototypes/go-disruptor)

# 待做事项

> gnet 还在持续开发的过程中，所以这个仓库的代码和文档会一直持续更新，如果你对 gnet 感兴趣的话，欢迎给这个开源库贡献你的代码，还有你要是喜欢 gnet 的话，可以给个星星鼓励一下哦 ~~