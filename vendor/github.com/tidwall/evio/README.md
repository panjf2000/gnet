<p align="center">
<img 
    src="logo.png" 
    width="213" height="75" border="0" alt="evio">
<br>
<a href="https://travis-ci.org/tidwall/evio"><img src="https://img.shields.io/travis/tidwall/evio.svg?style=flat-square" alt="Build Status"></a>
<a href="https://godoc.org/github.com/tidwall/evio"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>
</p>

`evio` is an event loop networking framework that is fast and small. It makes direct [epoll](https://en.wikipedia.org/wiki/Epoll) and [kqueue](https://en.wikipedia.org/wiki/Kqueue) syscalls rather than using the standard Go [net](https://golang.org/pkg/net/) package, and works in a similar manner as [libuv](https://github.com/libuv/libuv) and [libevent](https://github.com/libevent/libevent).

The goal of this project is to create a server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for packet handling. It was built to be the foundation for [Tile38](https://github.com/tidwall/tile38) and a future L7 proxy for Go.

*Please note: Evio should not be considered as a drop-in replacement for the standard Go net or net/http packages.*

## Features

- [Fast](#performance) single-threaded or [multithreaded](#multithreaded) event loop
- Built-in [load balancing](#load-balancing) options
- Simple API
- Low memory usage
- Supports tcp, [udp](#udp), and unix sockets
- Allows [multiple network binding](#multiple-addresses) on the same event loop
- Flexible [ticker](#ticker) event
- Fallback for non-epoll/kqueue operating systems by simulating events with the [net](https://golang.org/pkg/net/) package
- [SO_REUSEPORT](#so_reuseport) socket option

## Getting Started

### Installing

To start using evio, install Go and run `go get`:

```sh
$ go get -u github.com/tidwall/evio
```

This will retrieve the library.

### Usage

Starting a server is easy with `evio`. Just set up your events and pass them to the `Serve` function along with the binding address(es). Each connections is represented as an `evio.Conn` object that is passed to various events to differentiate the clients. At any point you can close a client or shutdown the server by return a `Close` or `Shutdown` action from an event.

Example echo server that binds to port 5000:

```go
package main

import "github.com/tidwall/evio"

func main() {
	var events evio.Events
	events.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
		out = in
		return
	}
	if err := evio.Serve(events, "tcp://localhost:5000"); err != nil {
		panic(err.Error())
	}
}
```

Here the only event being used is `Data`, which fires when the server receives input data from a client.
The exact same input data is then passed through the output return value, which is then sent back to the client. 

Connect to the echo server:

```sh
$ telnet localhost 5000
```

### Events

The event type has a bunch of handy events:

- `Serving` fires when the server is ready to accept new connections.
- `Opened` fires when a connection has opened.
- `Closed` fires when a connection has closed.
- `Detach` fires when a connection has been detached using the `Detach` return action.
- `Data` fires when the server receives new data from a connection.
- `Tick` fires immediately after the server starts and will fire again after a specified interval.

### Multiple addresses

A server can bind to multiple addresses and share the same event loop.

```go
evio.Serve(events, "tcp://192.168.0.10:5000", "unix://socket")
```

### Ticker

The `Tick` event fires ticks at a specified interval. 
The first tick fires immediately after the `Serving` events.

```go
events.Tick = func() (delay time.Duration, action Action){
	log.Printf("tick")
	delay = time.Second
	return
}
```

## UDP

The `Serve` function can bind to UDP addresses. 

- All incoming and outgoing packets are not buffered and sent individually.
- The `Opened` and `Closed` events are not availble for UDP sockets, only the `Data` event.

## Multithreaded

The `events.NumLoops` options sets the number of loops to use for the server. 
A value greater than 1 will effectively make the server multithreaded for multi-core machines. 
Which means you must take care when synchonizing memory between event callbacks. 
Setting to 0 or 1 will run the server as single-threaded. 
Setting to -1 will automatically assign this value equal to `runtime.NumProcs()`.

## Load balancing

The `events.LoadBalance` options sets the load balancing method. 
Load balancing is always a best effort to attempt to distribute the incoming connections between multiple loops.
This option is only available when `events.NumLoops` is set.

- `Random` requests that connections are randomly distributed.
- `RoundRobin` requests that connections are distributed to a loop in a round-robin fashion.
- `LeastConnections` assigns the next accepted connection to the loop with the least number of active connections.

## SO_REUSEPORT

Servers can utilize the [SO_REUSEPORT](https://lwn.net/Articles/542629/) option which allows multiple sockets on the same host to bind to the same port.

Just provide `reuseport=true` to an address:

```go
evio.Serve(events, "tcp://0.0.0.0:1234?reuseport=true"))
```

## More examples

Please check out the [examples](examples) subdirectory for a simplified [redis](examples/redis-server/main.go) clone, an [echo](examples/echo-server/main.go) server, and a very basic [http](examples/http-server/main.go) server.

To run an example:

```sh
$ go run examples/http-server/main.go
$ go run examples/redis-server/main.go
$ go run examples/echo-server/main.go
```

## Performance

### Benchmarks

These benchmarks were run on an ec2 c4.xlarge instance in single-threaded mode (GOMAXPROC=1) over Ipv4 localhost.
Check out [benchmarks](benchmarks) for more info.

<img src="benchmarks/out/echo.png" width="336" height="144" border="0" alt="echo benchmark"><img src="benchmarks/out/http.png" width="336" height="144" border="0" alt="http benchmark"><img src="benchmarks/out/redis_pipeline_1.png" width="336" height="144" border="0" alt="redis 1 benchmark"><img src="benchmarks/out/redis_pipeline_8.png" width="336" height="144" border="0" alt="redis 8 benchmark">


## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

`evio` source code is available under the MIT [License](/LICENSE).

