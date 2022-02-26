---
last_modified_on: "2022-02-26"
title: "Quickstart"
description: "High-level description of the gnet framework and its features."
---

## Installation

`gnet` is available as a Go module, with [Go 1.11 Modules](https://github.com/golang/go/wiki/Modules) support (Go 1.11+), just simply `import "github.com/panjf2000/gnet"` in your source code and `go [build|run|test]` will download the necessary dependencies automatically.

### V1
```powershell
go get -u github.com/panjf2000/gnet
```

### V2
```powershell
go get -u github.com/panjf2000/gnet/v2
```

## Example

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet/v2"
)

type echoServer struct {
	gnet.BuiltinEventEngine

	eng       gnet.Engine
	addr      string
	multicore bool
}

func (es *echoServer) OnBoot(eng gnet.Engine) gnet.Action {
	es.eng = eng
	log.Printf("echo server with multi-core=%t is listening on %s\n", es.multicore, es.addr)
	return gnet.None
}

func (es *echoServer) OnTraffic(c gnet.Conn) gnet.Action {
	buf, _ := c.Next(-1)
	c.Write(buf)
	return gnet.None
}

func main() {
	var port int
	var multicore bool

	// Example command: go run echo.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.Parse()
	echo := &echoServer{addr: fmt.Sprintf("tcp://:%d", port), multicore: multicore}
	log.Fatal(gnet.Run(echo, echo.addr, gnet.WithMulticore(multicore)))
}
```
As you can see, this example of echo server only sets up the `EventHandler.OnTraffic(Conn)` handler where you commonly write your main business code and this callback will be called as soon as the server receives network data from its peer, we call `Conn.Next(-1)` inside `OnTraffic()` to retrieve all data from a connection and then call `Conn.Write([]byte)` to send the data back to the peer via TCP socket, which make this an echo example, and finally we return a `gnet.None` under normal circumstances. If there is any unexpected error occuring, return either `gnet.Close` to close this connection or `gnet.Shutdown` to shut the whole server down.

There are two ways for users to retrieve data in `OnTraffic()`, one is to call `Conn.Peek(int)` and `Conn.Discard(int)`，the other is to call `Conn.Next(int)` , the former returns the next n bytes without advancing underlying connection buffer while the latter returns a slice containing the next n bytes from the buffer, advancing the buffer as if the bytes had been returned by `Conn.Read([]byte)`.

For more API's of `gnet.Conn`, please visit [gnet API Docs](https://pkg.go.dev/github.com/panjf2000/gnet).

