<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet" />
<br />
<a title="Build Status" target="_blank" href="https://github.com/panjf2000/gnet/actions?query=workflow%3ATests"><img src="https://img.shields.io/github/workflow/status/panjf2000/gnet/Tests?style=flat-square&logo=github-actions" /></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square&logo=codecov" /></a>
<a title="Supported Platforms" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/platform-Linux%20%7C%20FreeBSD%20%7C%20DragonFly%20%7C%20Darwin-549688?style=flat-square&logo=launchpad" /></a>
<a title="Require Go Version" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/go-%3E%3D1.9-30dff3?style=flat-square&logo=go" /></a>
<br />
<a title="Chat Room" target="_blank" href="https://gitter.im/gnet-io/gnet?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=body_badge"><img src="https://badges.gitter.im/gnet-io/gnet.svg" /></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square" /></a>
<a title="Doc for gnet" target="_blank" href="https://pkg.go.dev/github.com/panjf2000/gnet/v2#section-documentation"><img src="https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs" /></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go#networking"><img src="https://awesome.re/mentioned-badge-flat.svg" /></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/v/release/panjf2000/gnet.svg?color=161823&style=flat-square&logo=smartthings" /></a>
<a title="Tag" target="_blank" href="https://github.com/panjf2000/gnet/tags"><img src="https://img.shields.io/github/v/tag/panjf2000/gnet?color=%23ff8936&logo=fitbit&style=flat-square" /></a>
</p>

[英文](README.md) | 中文

# 📖 简介

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。它直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Go 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[netty](https://github.com/netty/netty) 和 [libuv](https://github.com/libuv/libuv)，这也使得 `gnet` 达到了一个远超 Go [net](https://golang.org/pkg/net/) 的性能表现。

`gnet` 设计开发的初衷不是为了取代 Go 的标准网络库：[net](https://golang.org/pkg/net/)，而是为了创造出一个类似于 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 能高效处理网络包的 Go 语言网络客户端/服务器框架。因此，`gnet` 在功能上的全面性并不如 [net](https://golang.org/pkg/net/)，它只提供网络编程中最核心的功能和最精简的 APIs，而且 `gnet` 也并没有打算变成一个全功能、无所不包的网络库，因为我觉得 Go [net](https://golang.org/pkg/net/) 在这方面已经做得足够好了。

`gnet` 的卖点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix Domain Socket）网络框架，开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但拥有更丰富的功能特性，且性能远胜之。**

# 🚀 功能

- [x] [高性能](#-性能测试) 的基于多线程/Go程网络模型的 event-loop 事件驱动
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 整个生命周期是无锁的
- [x] 简单易用的 APIs
- [x] 高效、可重用而且自动伸缩的内存 buffer：(Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] 支持多种网络协议/IPC 机制：`TCP`、`UDP` 和 `Unix Domain Socket`
- [x] 支持多种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`
- [x] 支持两种事件驱动机制：**Linux** 里的 `epoll` 以及 **FreeBSD/DragonFly/Darwin** 里的 `kqueue`
- [x] 灵活的事件定时器
- [x] 实现 `gnet` 客户端

# 🎬 开始

`gnet` 是一个 Go module，而且我们也强烈推荐通过 [Go Modules](https://go.dev/blog/using-go-modules) 来使用 `gnet`，在开启 Go Modules 支持（Go 1.11+）之后可以通过简单地在代码中写 `import "github.com/panjf2000/gnet"` 来引入 `gnet`，然后执行 `go mod download/go mod tidy` 或者 `go [build|run|test]` 这些命令来自动下载所依赖的包。

## 使用 v2 

```powershell
go get -u github.com/panjf2000/gnet/v2
```

## 使用 v1

```powershell
go get -u github.com/panjf2000/gnet
```

# 🎡 用户案例

以下公司/组织在生产环境上使用了 `gnet` 作为底层网络服务。

<a href="https://www.tencent.com"><img src="http://img.taohuawu.club/gallery/tencent_logo.png" width="250" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.iqiyi.com" target="_blank"><img src="http://img.taohuawu.club/gallery/iqiyi-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.mi.com" target="_blank"><img src="http://img.taohuawu.club/gallery/mi-logo.png" width="150" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.360.com" target="_blank"><img src="http://img.taohuawu.club/gallery/360-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://tieba.baidu.com/" target="_blank"><img src="http://img.taohuawu.club/gallery/baidu-tieba-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://game.qq.com/" target="_blank"><img src="http://img.taohuawu.club/gallery/tencent-games-logo.jpeg" width="200" align="middle"/></a>

如果你的项目也在使用 `gnet`，欢迎给我提 Pull Request 来更新这份列表。

# 📊 性能测试

## TechEmpower 性能测试

```powershell
# 硬件环境
CPU: 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
Mem: 32GB RAM
OS : Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
Net: Switched 10-gigabit ethernet
Go : go1.14.x linux/amd64
```

![All language](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-all.jpg)

这是包含全部编程语言框架的性能排名***前 50*** 的结果，总榜单包含了全世界共计 ***422 个框架***，其中 `gnet` 排名***第二***。


![Golang](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-go.png)

这是 Go 语言分类下的全部排名，`gnet` 超越了其他所有框架，位列第一，是***最快***的 Go 网络框架。

完整的排行可以通过 [TechEmpower Plaintext Benchmark](https://www.techempower.com/benchmarks/#section=test&runid=53c6220a-e110-466c-a333-2e879fea21ad&hw=ph&test=plaintext) 查看。

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

# ⚠️ 证书

`gnet` 的源码需在遵循 Apache-2.0 开源证书的前提下使用。

# 👏 贡献者

请在提 PR 之前仔细阅读 [Contributing Guidelines](CONTRIBUTING.md)，感谢那些为 `gnet` 贡献过代码的开发者！

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

# ⚓ 相关文章

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoller 原生网络模型之源码全面揭秘](https://strikefreedom.top/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://strikefreedom.top/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://strikefreedom.top/releasing-gnet-v1-with-techempower)

# 💰 支持

如果有意向，可以通过每个月定量的少许捐赠来支持这个项目。

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg"></a>

# 💎 赞助

每月定量捐赠 10 刀即可成为本项目的赞助者，届时您的 logo 或者 link 可以展示在本项目的 README 上。

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg"></a>

# ☕️ 打赏

> 当您通过以下方式进行捐赠时，请务必留下姓名、Github账号或其他社交媒体账号，以便我将其添加到捐赠者名单中，以表谢意。

<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" align="middle"/></a>&nbsp;&nbsp;

# 💴 资助者

<a target="_blank" href="https://github.com/patrick-othmer"><img src="https://avatars1.githubusercontent.com/u/8964313" width="100" alt="Patrick Othmer" /></a>&nbsp;<a target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://avatars2.githubusercontent.com/u/50285334" width="100" alt="Jimmy" /></a>&nbsp;<a target="_blank" href="https://github.com/cafra"><img src="https://avatars0.githubusercontent.com/u/13758306" width="100" alt="ChenZhen" /></a>&nbsp;<a target="_blank" href="https://github.com/yangwenmai"><img src="https://avatars0.githubusercontent.com/u/1710912" width="100" alt="Mai Yang" /></a>&nbsp;<a target="_blank" href="https://github.com/BeijingWks"><img src="https://avatars3.githubusercontent.com/u/33656339" width="100" alt="王开帅" /></a>&nbsp;<a target="_blank" href="https://github.com/refs"><img src="https://avatars3.githubusercontent.com/u/6905948" width="100" alt="Unger Alejandro" /></a>&nbsp;<a target="_blank" href="https://github.com/Swaggadan"><img src="https://avatars.githubusercontent.com/u/137142" width="100" alt="Swaggadan" /></a>&nbsp;<a target="_blank" href="https://github.com/Wuvist"><img src="https://avatars.githubusercontent.com/u/657796" width="100" alt="Weng Wei" /></a>

# 🔑 JetBrains 开源证书支持

`gnet` 项目一直以来都是在 JetBrains 公司旗下的 GoLand 集成开发环境中进行开发，基于 **free JetBrains Open Source license(s)** 正版免费授权，在此表达我的谢意。

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>

# 🔋 赞助商

<p>
	<h3>本项目由以下机构赞助：</h3>
	<a href="https://www.digitalocean.com/"><img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/SVG/DO_Logo_horizontal_blue.svg" width="201px" />
	</a>
</p>