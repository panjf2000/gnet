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

English | [中文](README_ZH.md)

# 📖 Introduction

`gnet` is an event-driven networking framework that is fast and lightweight. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package and works in a similar manner as [netty](https://github.com/netty/netty) and [libuv](https://github.com/libuv/libuv), which makes `gnet` achieve a much higher performance than Go [net](https://golang.org/pkg/net/).

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking client/server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling (although it does not limit itself to these areas), therefore, `gnet` is not as comprehensive as Go [net](https://golang.org/pkg/net/), it only provides the core functionalities (by a concise API set) of a networking application and it is not planned on being a full-featured networking framework, as I think [net](https://golang.org/pkg/net/) has done a good enough job in this area.

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

**`gnet` derives from the project: `evio` while having a much higher performance and more features.**

# 🚀 Features

- [x] [High-performance](#-performance) event-loop under networking model of multiple threads/goroutines
- [x] Built-in goroutine pool powered by the library [ants](https://github.com/panjf2000/ants)
- [x] Lock-free during the entire runtime
- [x] Concise and easy-to-use APIs
- [x] Efficient, reusable and elastic memory buffer: (Elastic-)Ring-Buffer, Linked-List-Buffer and Elastic-Mixed-Buffer
- [x] Supporting multiple protocols/IPC mechanism: `TCP`, `UDP` and `Unix Domain Socket`
- [x] Supporting multiple load-balancing algorithms: `Round-Robin`, `Source-Addr-Hash` and `Least-Connections`
- [x] Supporting two event-driven mechanisms: `epoll` on **Linux** and `kqueue` on **FreeBSD/DragonFly/Darwin**
- [x] Flexible ticker event
- [x] Implementation of `gnet` Client

# 🎬 Getting started

`gnet` is available as a Go module and we highly recommend that you use `gnet` via [Go Modules](https://go.dev/blog/using-go-modules), with Go 1.11 Modules enabled (Go 1.11+), you can just simply add `import "github.com/panjf2000/gnet"` to the codebase and run `go mod download/go mod tidy` or `go [build|run|test]` to download the necessary dependencies automatically.

## With v2 

```powershell
go get -u github.com/panjf2000/gnet/v2
```

## With v1

```powershell
go get -u github.com/panjf2000/gnet
```

# 🎡 Use cases

The following companies/organizations use `gnet` as the underlying network service in production.

<a href="https://www.tencent.com"><img src="http://img.taohuawu.club/gallery/tencent_logo.png" width="250" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.iqiyi.com" target="_blank"><img src="http://img.taohuawu.club/gallery/iqiyi-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.mi.com" target="_blank"><img src="http://img.taohuawu.club/gallery/mi-logo.png" width="150" align="middle"/></a>&nbsp;&nbsp;<a href="https://www.360.com" target="_blank"><img src="http://img.taohuawu.club/gallery/360-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://tieba.baidu.com/" target="_blank"><img src="http://img.taohuawu.club/gallery/baidu-tieba-logo.png" width="200" align="middle"/></a>&nbsp;&nbsp;<a href="https://game.qq.com/" target="_blank"><img src="http://img.taohuawu.club/gallery/tencent-games-logo.jpeg" width="200" align="middle"/></a>

If you have `gnet` integrated into projects, feel free to open a pull request refreshing this list.

# 📊 Performance

## Benchmarks on TechEmpower

```powershell
# Hardware Environment
CPU: 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
Mem: 32GB RAM
OS : Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
Net: Switched 10-gigabit ethernet
Go : go1.14.x linux/amd64
```

![All language](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-all.jpg)

This is the ***top 50*** on the framework ranking of all programming languages consists of a total of ***422 frameworks*** from all over the world where `gnet` is the ***runner-up***.


![Golang](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-go.png)

This is the full framework ranking of Go and `gnet` tops all the other frameworks, which makes `gnet` the ***fastest*** networking framework in Go.

To see the full ranking list, visit [TechEmpower Plaintext Benchmark](https://www.techempower.com/benchmarks/#section=test&runid=53c6220a-e110-466c-a333-2e879fea21ad&hw=ph&test=plaintext).

## Contrasts to the similar networking libraries

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

# ⚠️ License

Source code of `gnet` should be distributed under the Apache-2.0 license.

# 👏 Contributors

Please read the [Contributing Guidelines](CONTRIBUTING.md) before opening a PR and thank you to all the developers who already made contributions to `gnet`!

[![](https://opencollective.com/gnet/contributors.svg?width=890&button=false)](https://github.com/panjf2000/gnet/graphs/contributors)

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

Become a bronze sponsor with a monthly donation of $10 and get your logo on our README on Github.

<a href="https://opencollective.com/gnet#sponsors" target="_blank"><img src="https://opencollective.com/gnet/sponsors.svg"></a>

# ☕️ Buy me a coffee

> Please be sure to leave your name, Github account or other social media accounts when you donate by the following means so that I can add it to the list of donors as a token of my appreciation.

<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/WeChatPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/AliPay.JPG" width="250" align="middle"/>&nbsp;&nbsp;
<a href="https://www.paypal.me/R136a1X" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/payments/PayPal.JPG" width="250" align="middle"/></a>&nbsp;&nbsp;

# 💴 Patrons

<a target="_blank" href="https://github.com/patrick-othmer"><img src="https://avatars1.githubusercontent.com/u/8964313" width="100" alt="Patrick Othmer" /></a>&nbsp;<a target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://avatars2.githubusercontent.com/u/50285334" width="100" alt="Jimmy" /></a>&nbsp;<a target="_blank" href="https://github.com/cafra"><img src="https://avatars0.githubusercontent.com/u/13758306" width="100" alt="ChenZhen" /></a>&nbsp;<a target="_blank" href="https://github.com/yangwenmai"><img src="https://avatars0.githubusercontent.com/u/1710912" width="100" alt="Mai Yang" /></a>&nbsp;<a target="_blank" href="https://github.com/BeijingWks"><img src="https://avatars3.githubusercontent.com/u/33656339" width="100" alt="王开帅" /></a>&nbsp;<a target="_blank" href="https://github.com/refs"><img src="https://avatars3.githubusercontent.com/u/6905948" width="100" alt="Unger Alejandro" /></a>&nbsp;<a target="_blank" href="https://github.com/Swaggadan"><img src="https://avatars.githubusercontent.com/u/137142" width="100" alt="Swaggadan" /></a>&nbsp;<a target="_blank" href="https://github.com/Wuvist"><img src="https://avatars.githubusercontent.com/u/657796" width="100" alt="Weng Wei" /></a>

# 🔑 JetBrains OS licenses

`gnet` had been being developed with `GoLand` IDE under the **free JetBrains Open Source license(s)** granted by JetBrains s.r.o., hence I would like to express my thanks here.

<a href="https://www.jetbrains.com/?from=gnet" target="_blank"><img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/jetbrains/jetbrains-variant-4.png" width="250" align="middle"/></a>

# 🔋 Sponsorship

<p>
	<h3>This project is supported by:</h3>
	<a href="https://www.digitalocean.com/"><img src="https://opensource.nyc3.cdn.digitaloceanspaces.com/attribution/assets/SVG/DO_Logo_horizontal_blue.svg" width="201px" />
	</a>
</p>