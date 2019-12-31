<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/logos/master/gnet/logo.png" alt="gnet">
<br />
<a title="Build Status" target="_blank" href="https://travis-ci.com/panjf2000/gnet"><img src="https://img.shields.io/travis/com/panjf2000/gnet?style=flat-square&logo=appveyor"></a>
<a title="Codecov" target="_blank" href="https://codecov.io/gh/panjf2000/gnet"><img src="https://img.shields.io/codecov/c/github/panjf2000/gnet?style=flat-square&logo=appveyor"></a>
<a title="Supported Platforms" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-549688?style=flat-square&logo=appveyor"></a>
<a title="Require Go Version" target="_blank" href="https://github.com/panjf2000/gnet"><img src="https://img.shields.io/badge/go-%3E%3D1.9-30dff3?style=flat-square&logo=appveyor"></a>
<a title="Release" target="_blank" href="https://github.com/panjf2000/gnet/releases"><img src="https://img.shields.io/github/release/panjf2000/gnet.svg?color=161823&style=flat-square&logo=appveyor"></a>
<br/>
<a title="" target="_blank" href="https://golangci.com/r/github.com/panjf2000/gnet"><img src="https://golangci.com/badges/github.com/panjf2000/gnet.svg"></a>
<a title="Go Report Card" target="_blank" href="https://goreportcard.com/report/github.com/panjf2000/gnet"><img src="https://goreportcard.com/badge/github.com/panjf2000/gnet?style=flat-square"></a>
<a title="Doc for gnet" target="_blank" href="https://gowalker.org/github.com/panjf2000/gnet?lang=zh-CN"><img src="https://img.shields.io/badge/api-reference-8d4bbb.svg?style=flat-square&logo=appveyor"></a>
<a title="gnet on Sourcegraph" target="_blank" href="https://sourcegraph.com/github.com/panjf2000/gnet?badge"><img src="https://sourcegraph.com/github.com/panjf2000/gnet/-/badge.svg?style=flat-square"></a>
<a title="Mentioned in Awesome Go" target="_blank" href="https://github.com/avelino/awesome-go#networking"><img src="https://awesome.re/mentioned-badge-flat.svg"></a>
</p>

[英文](README.md) | 🇨🇳中文

# 📖 简介

`gnet` 是一个基于事件驱动的高性能和轻量级网络框架。它直接使用 [epoll](https://en.wikipedia.org/wiki/Epoll) 和 [kqueue](https://en.wikipedia.org/wiki/Kqueue) 系统调用而非标准 Golang 网络包：[net](https://golang.org/pkg/net/) 来构建网络应用，它的工作原理类似两个开源的网络库：[netty](https://github.com/netty/netty) 和 [libuv](https://github.com/libuv/libuv)。

这个项目存在的价值是提供一个在网络包处理方面能和 [Redis](http://redis.io)、[Haproxy](http://www.haproxy.org) 这两个项目具有相近性能的 Go 语言网络服务器框架。

`gnet` 的亮点在于它是一个高性能、轻量级、非阻塞的纯 Go 实现的传输层（TCP/UDP/Unix-Socket）网络框架，开发者可以使用 `gnet` 来实现自己的应用层网络协议(HTTP、RPC、Redis、WebSocket 等等)，从而构建出自己的应用层网络应用：比如在 `gnet` 上实现 HTTP 协议就可以创建出一个 HTTP 服务器 或者 Web 开发框架，实现 Redis 协议就可以创建出自己的 Redis 服务器等等。

**`gnet` 衍生自另一个项目：`evio`，但性能远胜之且拥有更丰富的功能特性。**

# 🚀 功能

- [x] [高性能](#-性能测试) 的基于多线程/Go程网络模型的 event-loop 事件驱动
- [x] 内置 Round-Robin 轮询负载均衡算法
- [x] 内置 goroutine 池，由开源库 [ants](https://github.com/panjf2000/ants) 提供支持
- [x] 内置 bytes 内存池，由开源库 [bytebufferpool](https://github.com/valyala/bytebufferpool) 提供支持
- [x] 简洁的 APIs
- [x] 基于 Ring-Buffer 的高效内存利用
- [x] 支持多种网络协议：TCP、UDP、Unix Sockets
- [x] 支持两种事件驱动机制：Linux 里的 epoll 以及 FreeBSD 里的 kqueue
- [x] 支持异步写操作
- [x] 灵活的事件定时器
- [x] SO_REUSEPORT 端口重用
- [x] 内置多种编解码器，支持对 TCP 数据流分包：LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec 和 LengthFieldBasedFrameCodec，参考自 [netty codec](https://github.com/netty/netty/tree/netty-4.1.43.Final/codec/src/main/java/io/netty/handler/codec)，而且支持自定制编解码器
- [x] 支持 Windows 平台，基于 ~~IOCP 事件驱动机制~~ Go 标准网络库
- [ ] 加入更多的负载均衡算法：随机、最少连接、一致性哈希等等
- [ ] 支持 TLS
- [ ] 实现 `gnet` 客户端

# 💡 核心设计
## 多线程/Go程网络模型
### 主从多 Reactors

`gnet` 重新设计开发了一个新内置的多线程/Go程网络模型：『主从多 Reactors』，这也是 `netty` 默认的多线程网络模型，下面是这个模型的原理图：

<p align="center">
<img alt="multi_reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img alt="reactor" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors-sequence-diagram.png">
</p>

### 主从多 Reactors + 线程/Go程池

你可能会问一个问题：如果我的业务逻辑是阻塞的，那么在 `EventHandler.React` 注册方法里的逻辑也会阻塞，从而导致阻塞 event-loop 线程，这时候怎么办？

正如你所知，基于 `gnet` 编写你的网络服务器有一条最重要的原则：永远不能让你业务逻辑（一般写在 `EventHandler.React` 里）阻塞 event-loop 线程，这也是 `netty` 的一条最重要的原则，否则的话将会极大地降低服务器的吞吐量。

我的回答是，基于`gnet` 的另一种多线程/Go程网络模型：『带线程/Go程池的主从多 Reactors』可以解决阻塞问题，这个新网络模型通过引入一个 worker pool 来解决业务逻辑阻塞的问题：它会在启动的时候初始化一个 worker pool，然后在把 `EventHandler.React`里面的阻塞代码放到 worker pool 里执行，从而避免阻塞 event-loop 线程。

模型的架构图如下所示：

<p align="center">
<img alt="multi_reactor_thread_pool" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool.png">
</p>

它的运行流程如下面的时序图：
<p align="center">
<img alt="multi-reactors" src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/multi-reactors%2Bthread-pool-sequence-diagram.png">
</p>

`gnet` 通过利用 [ants](https://github.com/panjf2000/ants) goroutine 池（一个基于 Go 开发的高性能的 goroutine 池 ，实现了对大规模 goroutines 的调度管理、goroutines 复用）来实现『主从多 Reactors + 线程/Go程池』网络模型。关于 `ants` 的全部功能和使用，可以在 [ants 文档](https://gowalker.org/github.com/panjf2000/ants?lang=zh-CN) 里找到。

`gnet` 内部集成了 `ants` 以及提供了 `pool.goroutine.Default()` 方法来初始化一个 `ants` goroutine 池，然后你可以把 `EventHandler.React` 中阻塞的业务逻辑提交到 goroutine 池里执行，最后在 goroutine 池里的代码调用 `gnet.Conn.AsyncWrite([]byte)` 方法把处理完阻塞逻辑之后得到的输出数据异步写回客户端，这样就可以避免阻塞 event-loop 线程。

有关在 `gnet` 里使用 `ants` goroutine 池的细节可以到[这里](#echo-server-with-blocking-logic)进一步了解。

## 自动扩容的 Ring-Buffer

`gnet` 内置了inbound 和 outbound 两个 buffers，基于 Ring-Buffer 原理实现，分别用来缓冲输入输出的网络数据以及管理内存。

对于 TCP 协议的流数据，使用 `gnet` 不需要业务方为了解析应用层协议而自己维护和管理 buffers，`gnet` 会替业务方完成缓冲和管理网络数据的任务，降低业务代码的复杂性以及降低开发者的心智负担，使得开发者能够专注于业务逻辑而非一些底层实现。

<p align="center">
<img src="https://raw.githubusercontent.com/panjf2000/illustrations/master/go/ring-buffer.gif">
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

用 `gnet` 来构建网络服务器是非常简单的，只需要实现 `gnet.EventHandler`接口然后把你关心的事件函数注册到里面，最后把它连同监听地址一起传递给 `gnet.Serve` 函数就完成了。在服务器开始工作之后，每一条到来的网络连接会在各个事件之间传递，如果你想在某个事件中关闭某条连接或者关掉整个服务器的话，直接在事件函数里把 `gnet.Action` 设置成 `Cosed` 或者 `Shutdown`就行了。

Echo 服务器是一种最简单网络服务器，把它作为 `gnet` 的入门例子在再合适不过了，下面是一个最简单的 echo server，它监听了 9000 端口：
### 不带阻塞逻辑的 echo 服务器

<details>
	<summary> Old version(<=v1.0.0-rc.4)  </summary>

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
</details>

```go
package main

import (
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
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

正如你所见，上面的例子里 `gnet` 实例只注册了一个 `EventHandler.React` 事件。一般来说，主要的业务逻辑代码会写在这个事件方法里，这个方法会在服务器接收到客户端写过来的数据之时被调用，然后处理输入数据（这里只是把数据 echo 回去）并且在处理完之后把需要输出的数据赋值给 `out` 变量然后返回，之后你就不用管了，`gnet` 会帮你把数据写回客户端的。

### 带阻塞逻辑的 echo 服务器

<details>
	<summary> Old version(<=v1.0.0-rc.4)  </summary>

```go
package main

import (
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type echoServer struct {
	*gnet.EventServer
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
	*gnet.EventServer
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

正如我在『主从多 Reactors + 线程/Go程池』那一节所说的那样，如果你的业务逻辑里包含阻塞代码，那么你应该把这些阻塞代码变成非阻塞的，比如通过把这部分代码通过 goroutine 去运行，但是要注意一点，如果你的服务器处理的流量足够的大，那么这种做法将会导致创建大量的 goroutines 极大地消耗系统资源，所以我一般建议你用 goroutine pool 来做 goroutines 的复用和管理，以及节省系统资源。

**各种 gnet 示例:**

<details>
	<summary> TCP Echo Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Echo server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}
func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	out = frame
	return

	/*
		// Echo asynchronously.
		data := append([]byte{}, frame...)
		go func() {
			time.Sleep(time.Second)
			c.AsyncWrite(data)
		}()
		return
	*/
}

func main() {
	var port int
	var multicore bool

	// Example command: go run echo.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.Parse()
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore)))
}
```
</details>

<details>
	<summary> UDP Echo Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("UDP Echo server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}
func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	out = frame
	return

	/*
		// Echo asynchronously.
		data := append([]byte{}, frame...)
		go func() {
			time.Sleep(time.Second)
			c.SendTo(data)
		}()
		return
	*/
}

func main() {
	var port int
	var multicore, reuseport bool

	// Example command: go run echo.go --port 9000 --multicore=true --reuseport=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.BoolVar(&reuseport, "reuseport", false, "--reuseport true")
	flag.Parse()
	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("udp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport)))
}
```
</details>

<details>
	<summary> HTTP Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/panjf2000/gnet"
)

var res string
var resBytes []byte

type request struct {
	proto, method string
	path, query   string
	head, body    string
	remoteAddr    string
}

type httpServer struct {
	*gnet.EventServer
}

var errMsg = "Internal Server Error"
var errMsgBytes = []byte(errMsg)

type httpCodec struct {
	req request
}

func (hc *httpCodec) Encode(c gnet.Conn, buf []byte) (out []byte, err error) {
	if c.Context() == nil {
		return appendHandle(out, res), nil
	}
	return appendResp(out, "500 Error", "", errMsg+"\n"), nil
}

func (hc *httpCodec) Decode(c gnet.Conn) ([]byte, error) {
	buf := c.Read()
	// process the pipeline
	leftover, err := parseReq(buf, &hc.req)
	// bad thing happened
	if err != nil {
		c.SetContext(err)
		return nil, err
	} else if len(leftover) == len(buf) {
		// request not ready, yet
		return nil, nil
	}
	c.ResetBuffer()
	return buf, nil
}

func (hs *httpServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("HTTP server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}

func (hs *httpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// process the pipeline
	if c.Context() != nil {
		// bad thing happened
		out = errMsgBytes
		action = gnet.Close
		return
	}
	// handle the request
	out = resBytes
	return
}

func main() {
	var port int
	var multicore bool

	// Example command: go run http.go --port 8080 --multicore=true
	flag.IntVar(&port, "port", 8080, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	res = "Hello World!\r\n"
	resBytes = []byte(res)

	http := new(httpServer)
	hc := new(httpCodec)

	// Start serving!
	log.Fatal(gnet.Serve(http, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithCodec(hc)))
}

// appendHandle handles the incoming request and appends the response to
// the provided bytes, which is then returned to the caller.
func appendHandle(b []byte, res string) []byte {
	return appendResp(b, "200 OK", "", res)
}

// appendResp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func appendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: gnet\r\n"...)
	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	b = append(b, head...)
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// parseReq is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parseReq(data []byte, req *request) (leftover []byte, err error) {
	sdata := string(data)
	var i, s int
	var head string
	var clen int
	var q = -1
	// method, path, proto line
	for ; i < len(sdata); i++ {
		if sdata[i] == ' ' {
			req.method = sdata[s:i]
			for i, s = i+1, i+1; i < len(sdata); i++ {
				if sdata[i] == '?' && q == -1 {
					q = i - s
				} else if sdata[i] == ' ' {
					if q != -1 {
						req.path = sdata[s:q]
						req.query = req.path[q+1 : i]
					} else {
						req.path = sdata[s:i]
					}
					for i, s = i+1, i+1; i < len(sdata); i++ {
						if sdata[i] == '\n' && sdata[i-1] == '\r' {
							req.proto = sdata[s:i]
							i, s = i+1, i+1
							break
						}
					}
					break
				}
			}
			break
		}
	}
	if req.proto == "" {
		return data, fmt.Errorf("malformed request")
	}
	head = sdata[:s]
	for ; i < len(sdata); i++ {
		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
			line := sdata[s : i-1]
			s = i + 1
			if line == "" {
				req.head = sdata[len(head)+2 : i+1]
				i++
				if clen > 0 {
					if len(sdata[i:]) < clen {
						break
					}
					req.body = sdata[i : i+clen]
					i += clen
				}
				return data[i:], nil
			}
			if strings.HasPrefix(line, "Content-Length:") {
				n, err := strconv.ParseInt(strings.TrimSpace(line[len("Content-Length:"):]), 10, 64)
				if err == nil {
					clen = int(n)
				}
			}
		}
	}
	// not enough data
	return data, nil
}
```
</details>

<details>
	<summary> Push Server </summary>

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/panjf2000/gnet"
)

type pushServer struct {
	*gnet.EventServer
	tick             time.Duration
	connectedSockets sync.Map
}

func (ps *pushServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Push server is listening on %s (multi-cores: %t, loops: %d), "+
		"pushing data every %s ...\n", srv.Addr.String(), srv.Multicore, srv.NumLoops, ps.tick.String())
	return
}
func (ps *pushServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Printf("Socket with addr: %s has been opened...\n", c.RemoteAddr().String())
	ps.connectedSockets.Store(c.RemoteAddr().String(), c)
	return
}
func (ps *pushServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Printf("Socket with addr: %s is closing...\n", c.RemoteAddr().String())
	ps.connectedSockets.Delete(c.RemoteAddr().String())
	return
}
func (ps *pushServer) Tick() (delay time.Duration, action gnet.Action) {
	log.Println("It's time to push data to clients!!!")
	ps.connectedSockets.Range(func(key, value interface{}) bool {
		addr := key.(string)
		c := value.(gnet.Conn)
		c.AsyncWrite([]byte(fmt.Sprintf("heart beating to %s\n", addr)))
		return true
	})
	delay = ps.tick
	return
}
func (ps *pushServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	out = frame
	return
}

func main() {
	var port int
	var multicore bool
	var interval time.Duration
	var ticker bool

	// Example command: go run push.go --port 9000 --tick 1s --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.DurationVar(&interval, "tick", 0, "pushing tick")
	flag.Parse()
	if interval > 0 {
		ticker = true
	}
	push := &pushServer{tick: interval}
	log.Fatal(gnet.Serve(push, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithTicker(ticker)))
}
```
</details>

<details>
	<summary> Codec Client/Server </summary>

**Client:**

```go
// Reference https://github.com/smallnest/goframe/blob/master/_examples/goclient/client.go

package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/smallnest/goframe"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	encoderConfig := goframe.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}

	decoderConfig := goframe.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	fc := goframe.NewLengthFieldBasedFrameConn(encoderConfig, decoderConfig, conn)
	err = fc.WriteFrame([]byte("hello"))
	if err != nil {
		panic(err)
	}
	err = fc.WriteFrame([]byte("world"))
	if err != nil {
		panic(err)
	}

	buf, err := fc.ReadFrame()
	if err != nil {
		panic(err)
	}
	fmt.Println("received: ", string(buf))
	buf, err = fc.ReadFrame()
	if err != nil {
		panic(err)
	}
	fmt.Println("received: ", string(buf))
}
```

**Server:**

```go
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type codecServer struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func (cs *codecServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Test codec server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumLoops)
	return
}

func (cs *codecServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if cs.async {
		data := append([]byte{}, frame...)
		_ = cs.workerPool.Submit(func() {
			c.AsyncWrite(data)
		})
		return
	}
	out = frame
	return
}

func testCodecServe(addr string, multicore, async bool, codec gnet.ICodec) {
	var err error
	if codec == nil {
		encoderConfig := gnet.EncoderConfig{
			ByteOrder:                       binary.BigEndian,
			LengthFieldLength:               4,
			LengthAdjustment:                0,
			LengthIncludesLengthFieldLength: false,
		}
		decoderConfig := gnet.DecoderConfig{
			ByteOrder:           binary.BigEndian,
			LengthFieldOffset:   0,
			LengthFieldLength:   4,
			LengthAdjustment:    0,
			InitialBytesToStrip: 4,
		}
		codec = gnet.NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	}
	cs := &codecServer{addr: addr, multicore: multicore, async: async, codec: codec, workerPool: goroutine.Default()}
	err = gnet.Serve(cs, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(codec))
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	var multicore bool

	// Example command: go run server.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()
	addr := fmt.Sprintf("tcp://:%d", port)
	testCodecServe(addr, multicore, false, nil)
}
```
</details>

**更详细的代码在这里: [gnet 示例](https://github.com/panjf2000/gnet/tree/master/examples)。**

## I/O 事件

 `gnet` 目前支持的 I/O 事件如下：

- `EventHandler.OnInitComplete` 当 server 初始化完成之后调用。
- `EventHandler.OnOpened` 当连接被打开的时候调用。
- `EventHandler.OnClosed` 当连接被关闭的之后调用。
- `EventHandler.React` 当 server 端接收到从 client 端发送来的数据的时候调用。（你的核心业务代码一般是写在这个方法里）
- `EventHandler.Tick` 服务器启动的时候会调用一次，之后就以给定的时间间隔定时调用一次，是一个定时器方法。
- `EventHandler.PreWrite` 预先写数据方法，在 server 端写数据回 client 端之前调用。


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

`gnet` 支持 UDP 协议，在 `gnet.Serve` 里绑定 UDP 地址即可，`gnet` 的 UDP 支持有如下的特性：

- 数据进入服务器之后立刻写回客户端，不做缓存。
-  `EventHandler.OnOpened` 和 `EventHandler.OnClosed` 这两个事件在 UDP 下不可用，唯一可用的事件是 `React`。
-  TCP 里的读写操作是 `Read()/ReadFrame()` 和 `AsyncWrite([]byte)` 方法，而在 UDP 里对应的方法是 `ReadFromUDP()` 和 `SendTo([]byte)`。

## 使用多核

`gnet.WithMulticore(true)` 参数指定了 `gnet` 是否会使用多核来进行服务，如果是 `true` 的话就会使用多核，否则就是单核运行，利用的核心数一般是机器的 CPU 数量。

## 负载均衡

`gnet` 目前内置的负载均衡算法是轮询调度 Round-Robin，暂时不支持自定制。

## SO_REUSEPORT 端口复用

服务器支持 [SO_REUSEPORT](https://lwn.net/Articles/542629/) 端口复用特性，允许多个 sockets 监听同一个端口，然后内核会帮你做好负载均衡，每次只唤醒一个 socket 来处理 accept 请求，避免惊群效应。

默认情况下，`gnet` 也不会有惊群效应，因为 `gnet` 默认的网络模型是主从多 Reactors，只会有一个主 reactor 在监听端口以及接受新连接。所以，开不开启 `SO_REUSEPORT` 选项是无关紧要的，只是开启了这个选项之后 `gnet` 的网络模型将会切换成 `evio` 的旧网络模型，这一点需要注意一下。

开启这个功能也很简单，使用 functional options 设置一下即可：

```go
gnet.Serve(events, "tcp://:9000", gnet.WithMulticore(true), gnet.WithReusePort(true)))
```

## 多种内置的 TCP 流编解码器

`gnet` 内置了多种用于 TCP 流分包的编解码器。

目前一共实现了 4 种常见的编解码器：LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec 和 LengthFieldBasedFrameCodec，基本上能满足大多数应用场景的需求了；而且 `gnet` 还允许用户实现自己的编解码器：只需要实现 [gnet.ICodec](https://github.com/panjf2000/gnet/blob/master/codec.go#L17) 接口，并通过 functional options 替换掉内部默认的编解码器即可。

这里有一个使用编解码器对 TCP 流分包的[例子](https://github.com/panjf2000/gnet/blob/master/examples/codec/server/server.go)。

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
- [bytebufferpool](https://github.com/valyala/bytebufferpool)
- [goframe](https://github.com/smallnest/goframe)

# 📚 相关文章

- [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
- [Going Infinite, handling 1M websockets connections in Go](https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go)
- [Go netpoll I/O 多路复用构建原生网络模型之源码深度解析](https://taohuawu.club/go-netpoll-io-multiplexing-reactor)
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