<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet" />
<br />
<a title="Build Status" target="_blank" href="https://github.com/panjf2000/gnet/actions?query=workflow%3ATests"><img src="https://img.shields.io/github/actions/workflow/status/panjf2000/gnet/test.yml?branch=dev&style=flat-square&logo=github-actions" /></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square&logo=codecov" /></a>
<a title="Supported Platforms" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20*BSD%20%7C%20Windows-549688?style=flat-square&logo=launchpad" /></a>
<a title="Minimum Go Version" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/go-%3E%3D1.20-30dff3?style=flat-square&logo=go" /></a>
<br />
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square" /></a>
<a title="Doc for gnet" target="_blank" href="https://pkg.go.dev/github.com/panjf2000/gnet/v2#section-documentation"><img src="https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs" /></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go#networking"><img src="https://awesome.re/mentioned-badge-flat.svg" /></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/v/release/panjf2000/gnet.svg?color=161823&style=flat-square&logo=smartthings" /></a>
<a title="Tag" target="_blank" href="https://github.com/panjf2000/gnet/tags"><img src="https://img.shields.io/github/v/tag/panjf2000/gnet?color=%23ff8936&logo=fitbit&style=flat-square" /></a>
</p>

<p align="center">
<a href="https://trendshift.io/repositories/9602" target="_blank"><img src="https://trendshift.io/api/badge/repositories/9602" alt="panjf2000%2Fgnet | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>
</p>

[英文](README.md) | 中文

### 🎉🎉🎉 欢迎加入 `gnet` 在 [Discord 服务器上的频道](https://discord.gg/UyKD7NZcfH).

# 📖 简介

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。这个框架是基于 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 从零开发的，而且相比 Go [net](https://golang.org/pkg/net/)，它能以更低的内存占用实现更高的性能。

`gnet` 和 [net](https://golang.org/pkg/net/) 有着不一样的网络编程范式。因此，用 `gnet` 开发网络应用和用 [net](https://golang.org/pkg/net/) 开发区别很大，而且两者之间不可调和。社区里有其他同类的产品像是 [libuv](https://github.com/libuv/libuv), [netty](https://github.com/netty/netty), [twisted](https://github.com/twisted/twisted), [tornado](https://github.com/tornadoweb/tornado)，`gnet` 的底层工作原理和这些框架非常类似。

`gnet` 不是为了取代 [net](https://golang.org/pkg/net/) 而生的，而是在 Go 生态中为开发者提供一个开发性能敏感的网络服务的替代品。也正因如此，`gnet` 在功能全面性上比不了 Go [net](https://golang.org/pkg/net/)，它只会提供网络应用所需的最核心的功能和最精简的 APIs，而且 `gnet` 也并没有打算变成一个无所不包的网络框架，因为我觉得 Go [net](https://golang.org/pkg/net/) 在这方面已经做得足够好了。

`gnet` 的卖点在于它是一个高性能、轻量级、非阻塞的纯 Go 语言实现的传输层（TCP/UDP/Unix Domain Socket）网络框架。开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络服务。比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但拥有更丰富的功能特性，且性能远胜之。**

# 🚀 功能

## 🦖 里程碑

- [x] 基于多线程/协程网络模型的[高性能](#-性能测试)事件驱动循环
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 整个生命周期是无锁的
- [x] 简单易用的 APIs
- [x] 高效、可重用而且自动伸缩的内存 buffer：(Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] 多种网络协议/IPC 机制：`TCP`、`UDP` 和 `Unix Domain Socket`
- [x] 多种负载均衡算法：`Round-Robin(轮询)`、`Source-Addr-Hash(源地址哈希)` 和 `Least-Connections(最少连接数)`
- [x] 灵活的事件定时器
- [x] `gnet` 客户端支持
- [x] 支持 `Linux`, `macOS`, `Windows` 和 *BSD 操作系统: `Darwin`/`DragonFlyBSD`/`FreeBSD`/`NetBSD`/`OpenBSD`
- [x] **Edge-triggered** I/O 支持
- [x] 多网络地址绑定
- [x] 支持注册新的连接到事件循环

## 🕊 蓝图

- [ ] 支持 **TLS**
- [ ] 支持 [io_uring](https://github.com/axboe/liburing/wiki/io_uring-and-networking-in-2023)
- [ ] 支持 **KCP**

***`gnet` 的 Windows 版本应该仅用于开发阶段的开发和测试，切勿用于生产环境***。

# 🎬 开始

`gnet` 是一个 Go module，而且我们也强烈推荐通过 [Go Modules](https://go.dev/blog/using-go-modules) 来使用 `gnet`，在开启 Go Modules 支持（Go 1.11+）之后可以通过简单地在代码中写 `import "github.com/panjf2000/gnet/v2"` 来引入 `gnet`，然后执行 `go mod download/go mod tidy` 或者 `go [build|run|test]` 这些命令来自动下载所依赖的包。

## 使用 v2

```bash
go get -u github.com/panjf2000/gnet/v2
```

## 使用 v1

```bash
go get -u github.com/panjf2000/gnet
```

# 🎡 用户案例

以下公司/组织在生产环境上使用了 `gnet` 作为底层网络服务。

<table>
  <tbody>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.tencent.com/">
          <img src="https://res.strikefreedom.top/static_res/logos/tencent_logo.png" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.tencentgames.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/tencent-games-logo.jpeg" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.iqiyi.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/iqiyi-logo.png" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.mi.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/mi-logo.png" width="200" />
        </a>
      </td>
    </tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.360.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/360-logo.png" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://tieba.baidu.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/baidu-tieba-logo.png" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.jd.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/jd-logo.png" width="200" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.zuoyebang.com/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/zuoyebang-logo.jpeg" width="200" />
        </a>
      </td>
    </tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.bytedance.com/zh/" target="_blank">
          <img src="https://res.strikefreedom.top/static_res/logos/ByteDance_Logo.png" width="250" />
        </a>
      </td>
    </tr>
  </tbody>
</table>

如果你也正在生产环境上使用 `gnet`，欢迎提 Pull Request 来丰富这份列表。

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

***请注意，TechEmpower 上的 gnet 的 HTTP 实现是不完备且针对性调优的，仅仅是用于压测目的，不是生产可用的***。

## 同类型的网络库性能对比

### On Linux (epoll)

#### Environment

```bash
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

[![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_linux.png)](https://github.com/gnet-io/gnet-benchmarks)

[![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_linux.png)]((https://github.com/gnet-io/gnet-benchmarks))

### On MacOS (kqueue)

#### Environment

```bash
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

[![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_macos.png)]((https://github.com/gnet-io/gnet-benchmarks))

[![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_macos.png)]((https://github.com/gnet-io/gnet-benchmarks))

### "硬刚" Rust

[![](https://res.strikefreedom.top/static_res/blog/figures/Gjfx2GoXAAA5haW.jpeg)](https://www.youtube.com/watch?v=31R8Ef9A0iw)

# ⚠️ 证书

`gnet` 的源码需在遵循 Apache-2.0 开源证书的前提下使用。

# 👏 贡献者

请在提 PR 之前仔细阅读 [Contributing Guidelines](CONTRIBUTING.md)，感谢那些为 `gnet` 贡献过代码的开发者！

<a href="https://github.com/panjf2000/gnet/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=panjf2000/gnet" />
</a>

# ⚓ 相关文章

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoller 原生网络模型之源码全面揭秘](https://strikefreedom.top/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://strikefreedom.top/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://strikefreedom.top/releasing-gnet-v1-with-techempower)

# ☕️ 打赏

> 当您通过以下方式进行捐赠时，请务必留下姓名、GitHub 账号或其他社交媒体账号，以便我将其添加到捐赠者名单中，以表谢意。

<table>
  <tbody>
    <tr>
      <td align="center" valign="middle">
        <a target="_blank" href="https://buymeacoffee.com/panjf2000">
          <img src="https://res.strikefreedom.top/static_res/logos/bmc_qr.png" width="250" alt="By me coffee" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a target="_blank" href="https://www.patreon.com/panjf2000">
          <img src="https://res.strikefreedom.top/static_res/logos/patreon_logo.png" width="250" alt="Patreon" />
        </a>
      </td>
      <td align="center" valign="middle">
        <a target="_blank" href="https://opencollective.com/panjf2000">
          <img src="https://res.strikefreedom.top/static_res/logos/open-collective-logo.png" width="250" alt="OpenCollective" />
        </a>
      </td>
    </tr>
  </tbody>
</table>

# 🔑 JetBrains 开源证书支持

`gnet` 项目一直以来都是在 JetBrains 公司旗下的 GoLand 集成开发环境中进行开发，基于 ***free JetBrains Open Source license(s)*** 正版免费授权，在此表达我的谢意。

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jetbrains.svg" alt="JetBrains logo."></a>

# 🔋 赞助商

[![DigitalOcean Referral Badge](https://web-platforms.sfo2.cdn.digitaloceanspaces.com/WWW/Badge%203.svg)](https://www.digitalocean.com/?refcode=5d8774f42124&utm_campaign=Referral_Invite&utm_medium=Referral_Program&utm_source=badge)
