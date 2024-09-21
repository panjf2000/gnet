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

English | [中文](README_ZH.md)

### 🎉🎉🎉 Feel free to join [the channels about `gnet` on the Discord Server](https://discord.gg/UyKD7NZcfH).

# 📖 Introduction

`gnet` is an event-driven networking framework that is ultra-fast and lightweight. It is built from scratch by exploiting [epoll](https://man7.org/linux/man-pages/man7/epoll.7.html) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) and it can achieve much higher performance with lower memory consumption than Go [net](https://golang.org/pkg/net/) in many specific scenarios.

`gnet` and [net](https://golang.org/pkg/net/) don't share the same philosophy about network programming. Thus, building network applications with `gnet` can be significantly different from building them with [net](https://golang.org/pkg/net/), and the philosophies can't be reconciled. There are other similar products written in other programming languages in the community, such as [libevent](https://github.com/libevent/libevent), [libuv](https://github.com/libuv/libuv), [netty](https://github.com/netty/netty), [twisted](https://github.com/twisted/twisted), [tornado](https://github.com/tornadoweb/tornado), etc. which work in a similar pattern as `gnet` under the hood.

`gnet` is not designed to displace the Go [net](https://golang.org/pkg/net/), but to create an alternative in the Go ecosystem for building performance-critical network services. As a result of which, `gnet` is not as comprehensive as Go [net](https://golang.org/pkg/net/), it provides only the core functionalities (in a concise API set) required by a network application and it doesn't plan on being a coverall networking framework, as I think Go [net](https://golang.org/pkg/net/) has done a good enough job in that area.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on the transport layer with TCP/UDP protocols and Unix Domain Socket. It enables developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building diversified network services. For instance, you get an HTTP Server if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` with much higher performance and more features.**

# 🚀 Features

## 🦖 Milestone

- [x] [High-performance](#-performance) event-driven looping based on a networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Lock-free during the entire runtime
- [x] Concise and easy-to-use APIs
- [x] Efficient, reusable, and elastic memory buffer: (Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] Multiple protocols/IPC mechanisms: `TCP`, `UDP`, and `Unix Domain Socket`
- [x] Multiple load-balancing algorithms: `Round-Robin`, `Source-Addr-Hash`, and `Least-Connections`
- [x] Flexible ticker event
- [x] `gnet` client
- [x] Running on `Linux`, `macOS`, `Windows`, and *BSD: `Darwin`/`DragonFlyBSD`/`FreeBSD`/`NetBSD`/`OpenBSD`
- [x] **Edge-triggered** I/O support
- [x] Multiple network addresses binding

## 🕊 Roadmap

- [ ] **TLS** support
- [ ] [io_uring](https://github.com/axboe/liburing/wiki/io_uring-and-networking-in-2023) support
- [ ] **KCP** support

***Windows version of `gnet` should only be used in development for developing and testing, it shouldn't be used in production.***

# 🎬 Getting started

`gnet` is available as a Go module and we highly recommend that you use `gnet` via [Go Modules](https://go.dev/blog/using-go-modules), with Go 1.11 Modules enabled (Go 1.11+), you can just simply add `import "github.com/panjf2000/gnet/v2"` to the codebase and run `go mod download/go mod tidy` or `go [build|run|test]` to download the necessary dependencies automatically.

## With v2

```bash
go get -u github.com/panjf2000/gnet/v2
```

## With v1

```bash
go get -u github.com/panjf2000/gnet
```

# 🎡 Use cases

The following corporations/organizations use `gnet` as the underlying network service in production.

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
        <a href="https://www.mi.com/global/" target="_blank">
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
  </tbody>
</table>

If you're also using `gnet` in production, please help us enrich this list by opening a pull request.

# 📊 Performance

## Benchmarks on TechEmpower

```bash
# Hardware Environment
* 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 3.20GHz
* 32GB RAM
* Dedicated Cisco 10-gigabit Ethernet switch
* Debian 12 "bookworm"
* Go1.19.x linux/amd64
```

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-top50-light.jpg)

This is a leaderboard of the top ***50*** out of ***486*** frameworks that encompass various programming languages worldwide, in which `gnet` is ranked ***first***.

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-topN-go-light.png)

This is the full framework ranking of Go and `gnet` tops all the other frameworks, which makes `gnet` the ***fastest*** networking framework in Go.

To see the full ranking list, visit [TechEmpower Benchmark **Round 22**](https://www.techempower.com/benchmarks/#hw=ph&test=plaintext&section=data-r22).

***Note that the HTTP implementation of gnet on TechEmpower is half-baked and fine-tuned for benchmark purposes only and far from production-ready.***

## Contrasts to the similar networking libraries

## On Linux (epoll)

### Test Environment

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

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_linux.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_linux.png)

## On MacOS (kqueue)

### Test Environment

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

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_macos.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_macos.png)

# ⚠️ License

The source code of `gnet` should be distributed under the Apache-2.0 license.

# 👏 Contributors

Please read the [Contributing Guidelines](CONTRIBUTING.md) before opening a PR and thank you to all the developers who already made contributions to `gnet`!

<a href="https://github.com/panjf2000/gnet/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=panjf2000/gnet" />
</a>

# ⚓ Relevant Articles

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoller 原生网络模型之源码全面揭秘](https://strikefreedom.top/go-netpoll-io-multiplexing-reactor)
- [gnet: 一个轻量级且高性能的 Golang 网络库](https://strikefreedom.top/go-event-loop-networking-library-gnet)
- [最快的 Go 网络框架 gnet 来啦！](https://strikefreedom.top/releasing-gnet-v1-with-techempower)

# 💰 Backers

Support us with a monthly donation and help us continue our activities.

<a href="https://opencollective.com/gnet#backers" target="_blank"><img src="https://opencollective.com/gnet/backers.svg"></a>

# 💎 Sponsors

Become a bronze sponsor with a monthly donation of $10 and get your logo on our README on GitHub.

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg"></a>

# ☕️ Buy me a coffee

> Please be sure to leave your name, GitHub account, or other social media accounts when you donate by the following means so that I can add it to the list of donors as a token of my appreciation.

<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" align="middle"/></a>&nbsp;&nbsp;

# 🔑 JetBrains OS licenses

`gnet` had been being developed with `GoLand` IDE under the **free JetBrains Open Source license(s)** granted by JetBrains s.r.o., hence I would like to express my thanks here.

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>

# 🔋 Sponsorship

<p>
  <h3>This project is supported by:</h3>
  <a href="https://www.digitalocean.com/"><img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/SVG/DO_Logo_horizontal_blue.svg" width="201px" />
  </a>
</p>