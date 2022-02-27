---
last_modified_on: "2022-02-26"
id: faq
title: "FAQ"
description: "Frequently asked questions about gnet."
---

## Architecture & Code & Principles

###  Why is gnet so fast?

gnet's networking model is a `Reactor` networking model with a event-driven mechanism that is designed and tuned to manage millions of network connections and handle a zillion requests, which backs gnet up to be the fatest networking framework in Go, and all it takes is a few goroutines.

In addition to the first-class networking model, the implementation of auto-scaling and reusable elastic buffers in gnet is also one of the critical essentials for its high performance.

..., etc.

## Stability

### Is gnet solid enough to be used in production?

Sure it is! 

Actually, there are already many companies/organizations using `gnet` as the underlying network service in production and they have been worked well for a long time.

Here is a partial list of gnet use cases: https://gnet.host/#usecases.

..., etc.

## Scope of application & usage

### When should I use gnet instead of the standard net in Go?

`gnet` sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go which works on transport layer with TCP/UDP protocols and Unix Domain Socket , so it allows developers to implement their own protocols(HTTP, RPC, WebSocket, Redis, etc.) of application layer upon `gnet` for building  diversified network applications, for instance, you get an HTTP Server or Web Framework if you implement HTTP protocol upon `gnet` while you have a Redis Server done with the implementation of Redis protocol upon `gnet` and so on.

`gnet` is not designed to displace the standard Go [net](https://golang.org/pkg/net/) package, but to create a networking client/server framework for Go that performs on par with [Redis](http://redis.io) and [Haproxy](http://www.haproxy.org) for networking packets handling (although it does not limit itself to these areas), therefore, `gnet` is not as comprehensive as Go [net](https://golang.org/pkg/net/), it only provides the core functionalities (by a concise API set) of a networking application and it is not planned on being a full-featured networking framework, as I think [net](https://golang.org/pkg/net/) has done a good enough job in this area.

In a word, if performance is not your top priority and you intend to take care of all corners during the networking development, then you should go with [net](https://golang.org/pkg/net/), but if you are trying to build a insanely fast networking application with a very low resource footprint and looking for a solution, I believe `gnet` is the right choice for you.

### How can I build networking applications of diverse protocols on top of gnet?

There are some examples powered by gnet framework, go check out those source code and get an initial perception about developing networking applications based on gnet, after that, you might want to read the documentations of gnet to learn all its APIs and try to write a demo application.

- [Examples for gnet](https://github.com/gnet-io/gnet-examples)
- [Documentations](https://gnet.host/docs/)

..., etc.

## Contributing

###  How do I contribute to gnet?

gnet is [open-source](https://github.com/panjf2000/gnet) and welcomes contributions. A few guidelines to help you get started:

1. Read our [contribution guide](https://github.com/panjf2000/gnet/blob/master/CONTRIBUTING.md).
2. Start with [good first issues](https://github.com/panjf2000/gnet/issues?q=is%3Aissue+label%3A"good+first+issue").
3. Join our [chat](https://gitter.im/gnet-io/gnet) if you have any questions. We are happy to help!
