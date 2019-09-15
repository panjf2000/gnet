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

`gnet` is an event-loop networking framework that is fast and small. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [libuv](https://github.com/libuv/libuv) and [libevent](https://github.com/libevent/libevent).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling.

`gnet` sells itself as a high-performance, lightweight, nonblocking network library written in pure Go.

`gent` is derived from project `evio` while having higher performance.

> gnet is still under active development, so if you are interested in gnet, please feel free to make your code contributions to it ~~

# Benchmark Test

## On Linux (epoll)

### Test Environment

```powershell
Go Version: go1.12.9 linux/amd64
OS:         Ubuntu 18.04
CPU:        8 Virtual CPUs
Memory:     16.0 GiB
```



### Echo Server

![](benchmarks/results/echo_linux.png)

### HTTP Server

![](benchmarks/results/http_linux.png)

## On FreeBSD (kqueue)

### Test Environment

```powershell
Go Version: go version go1.12.9 darwin/amd64
OS:         macOS Mojave 10.14.6
CPU:        4 CPUs
Memory:     8.0 GiB
```



### Echo Server

![](benchmarks/results/echo_mac.png)

### HTTP Server

![](benchmarks/results/http_mac.png)