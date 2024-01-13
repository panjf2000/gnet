---
id: best-practices
last_modified_on: "2022-09-24"
title: "Best practices"
description: "Best practices for writing code on top of gnet."
---

### Never run blocking code in OnTraffic(), OnOpen() and OnClose()

The above three event handlers (callbacks) are executed in event-loops, therefore, running blocking code in them blocks event-loops, which means that the subsequent tasks will have to wait for the preceding blocking event handlers to complete before they get executed.

To avoid blocking event-loops, asynchronize your blocking code, for example by starting a goroutine with your blocking code and invoking c.AsyncWrite() or c.AsyncWritev() to send response data to the peer endpoint.

If you're not familiar with how gnet works, go back and read [this](https://gnet.host/docs/about/overview/#networking-model-of-multiple-threadsgoroutines).

### Either loop read data in OnTraffic() or invoke c.Wake() regularly

Despite the fact that gnet leverages epoll/kqueue with level-triggered mode under the hood, OnTraffic() won't be invoked constantly given there is data left in the inbound buffer of a connection, OnTraffic() is only invoked when the connection has new data arriving, which is like edge-triggered mode from the user's point of view.

Thus, you should loop call c.Read()/c.Peek()/c.Next() for a connection in OnTraffic() to drain the inbound buffer of incoming data, but if you don't, then make sure you call c.Wake() periodically, otherwise you may never get a chance to read the rest of the data sent by the peer endpoint (client or server) unless the peer endpoint sends new data over.


### Leverage Conn.Context() to monopolize data instead of sharing it across connections

It's recommended to use Conn.Context() to store necessary resource for each connection, so that each connection can take advantage of its exclusive resource, avoiding the contention of single resource across connections.

### Enable poll_opt mode to boost performance

By default, `gnet` utilizes the standard package `golang.org/x/sys/unix` to implement pollers with `epoll` or `kqueue`, where a HASH map of `fd->conn` is introduced to help retrieve connections by file descriptors returned from pollers, but now you can run `go build` with build tags `poll_opt`, like this: `go build -tags=poll_opt`, and `gnet` will switch to the optimized implementations of pollers that invoke the system calls of `epoll` or `kqueue` directly and add file descriptors to the interest list along with storing the corresponding connection pointers into `epoll_data` or `kevent`, in which case `gnet` can get rid of the HASH MAP of `fd->conn` and regain each connection pointer by the conversion of `void*` pointer in the I/O event-looping. In theory, it ought to achieve a higher performance with this optimization.

Visit [#230](https://github.com/panjf2000/gnet/pull/230) for code details.

### Enable gc_opt mode to reduce GC latency

By default, `gnet` uses `map` as the internal storage of connections, but now you can run `go build` with build tags `gc_opt`, like this: `go build -tags=gc_opt`, and `gnet` will switch to the optimized implementation of connections storage that uses a new data structure `matrix` for managing connections, in which case `gnet` eliminates the pointers in `map` to reduce the GC latency significantly.

Visit [Announcing gnet v2.3.0](https://gnet.host/blog/announcing-gnet-v2-3-0/) for more details.

### To be continued