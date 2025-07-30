---
id: best-practices
last_modified_on: "2024-04-16"
title: "Best practices"
description: "Best practices for writing code on top of gnet."
---

### Never run blocking code in OnTraffic(), OnOpen() and OnClose()

The above three event handlers (callbacks) are executed in event-loops, therefore, running blocking code in them blocks event-loops, which means that the subsequent tasks will have to wait for the preceding blocking event handlers to complete before they get executed.

To avoid blocking event-loops, asynchronize your blocking code, for example by starting a goroutine with your blocking code and invoking `Conn.AsyncWrite()` or `Conn.AsyncWritev()` to send response data to the peer endpoint.

If you're not familiar with how `gnet` works, go back and read [this](https://gnet.host/docs/about/overview/#networking-model-of-multiple-threadsgoroutines).

### Avoid data corruption

Any incoming bytes returned by `Conn.Peek`/`Conn.Next` must not be used in a new goroutine and the buffer returned by `Conn.Peek` must not be used after calling `Conn.Discard`.

Otherwise, make a copy of the buffer returned by `Conn.Peek`/`Conn.Next` manually or call `Conn.Read()` to read the data into a new buffer to avoid data corruption.


### Leverage Conn.Context() to monopolize data instead of sharing it across connections

It's recommended to use `Conn.Context()` to store necessary resource for each connection, so that each connection can take advantage of its exclusive resource, avoiding the contention of single resource across connections.

### Either loop read data in OnTraffic() or invoke c.Wake() regularly

`gnet` leverages `epoll`/`kqueue` with level-triggered mode by default under the hood, you're able to switch to edge-triggered mode since v2.5.0. In LT mode, `OnTraffic()` might not be invoked constantly given there is data left in the inbound buffer of a `gnet.Conn`, `OnTraffic()` will be invoked only when there is data left in the socket recv buffer of the kernel. By contrast, in ET mode, `OnTraffic()` will be invoked only when new data arrives at the socket recv buffer of the kernel.

Thus, you should loop call `c.Read()`/`c.Peek()`/`c.Next()` on a connection in `OnTraffic()` to drain the inbound buffer for reading and decoding packets until you reach an incomplete packet, but if you don't, then make sure you call `c.Wake()` periodically, otherwise you may never get a chance to read the leftover data until the remote sends new data over and there are new arrivals of data on the socket recv buffer.

### Enable poll_opt mode to boost performance

By default, `gnet` utilizes the standard package `golang.org/x/sys/unix` to implement pollers with `epoll` or `kqueue`, where a HASH map of `fd->conn` is introduced to help retrieve connections by file descriptors returned from pollers, but now you can run `go build` with build tags `poll_opt`, like this: `go build -tags=poll_opt`, and `gnet` will switch to the optimized implementations of pollers that invoke the system calls of `epoll` or `kqueue` directly and add file descriptors to the interest list along with storing the corresponding connection pointers into `epoll_data` or `kevent`, in which case `gnet` can get rid of the HASH MAP of `fd->conn` and regain each connection pointer by the conversion of `void*` pointer in the I/O event-looping. In theory, it ought to achieve a higher performance with this optimization.

Visit [#230](https://github.com/panjf2000/gnet/pull/230) for code details.

### Enable gc_opt mode to reduce GC latency

By default, `gnet` uses `map` as the internal storage of connections, but now you can run `go build` with build tags `gc_opt`, like this: `go build -tags=gc_opt`, and `gnet` will switch to the optimized implementation of connections storage that uses a new data structure `matrix` for managing connections, in which case `gnet` eliminates the pointers in `map` to reduce the GC latency significantly.

Visit [Announcing gnet v2.3.0](https://gnet.host/blog/announcing-gnet-v2-3-0/) for more details.

### To be continued