---
last_modified_on: "2024-04-22"
id: announcing-gnet-v2-5-0
title: Announcing gnet v2.5.0
description: "Hello World! We present you, gnet v2.5.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

The v2.5.0 for `gnet` is officially released!

The two major updates in this release are [feat: support edge-triggered I/O](https://github.com/panjf2000/gnet/pull/576) and [feat: support multiple network addresses binding](https://github.com/panjf2000/gnet/pull/578).

## Intro

In [#576](https://github.com/panjf2000/gnet/pull/576), `gnet` implemented edge-triggered I/O on the basis of `EPOLLET` in `epoll` and `EV_CLEAR` in `kqueue`. Before v2.5.0, `gnet` has been using level-triggered I/O under the hood, now developers are able to switch to edge-triggered I/O via functional option: `EdgeTriggeredIO ` when developing and deploying `gnet` services. In certain specific scenarios, edge-triggered I/O may outperform level-triggered I/O, as a result of which, switching `gnet` from LT mode to ET mode can lead to significant performance improvements. But note that this performance boost is only theoretical inference and may only occur under specific circumstances. ***Therefore, please use ET mode with caution and conduct benchmark tests to collect sufficient numbers before the deployment in production.***

Another useful new feature is [#578](https://github.com/panjf2000/gnet/pull/578), with which developers are allowed to bind multiple addresses(IP:Port) in one `gnet` instance. This feature makes it possible to build and run a `gnet` server that serves for various protocols or a specific set of backend services.

In addition to these two major features, we've also made many optimizations to the project code, refactoring and streamlining the core code, as well as optimising the structure.

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!