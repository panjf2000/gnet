---
last_modified_on: "2026-07-03"
id: announcing-gnet-v2-10-0
title: Announcing gnet v2.10.0
description: "Hello World! We present you, gnet v2.10.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

![](/img/gnet-v2-10-0.jpg)

The `gnet` v2.10.0 is officially released!

***The most significant enhancement in this release is the support for reading and writing the connection-level context across goroutines!***

Previously, `Conn.Context()`/`Conn.SetContext()` could only be safely invoked within the `EventHandler` callbacks running on the connection's own event-loop goroutine. This made it inconvenient to share and update per-connection state from other goroutines, such as background workers or other event-loops.

Starting with v2.10.0, `gnet` introduces `Conn.SafeContext()`/`Conn.SetSafeContext()`, a concurrency-safe counterpart of `Context()`/`SetContext()` that can be invoked from any goroutine at any time:

```go
// SafeContext is the concurrency-safe version of Context that can
// be invoked on any goroutine.
SafeContext() (ctx any)

// SetSafeContext is the concurrency-safe version of SetContext that can
// be invoked on any goroutine.
SetSafeContext(ctx any)
```

This makes it much easier to build applications where other goroutines need to read or update the state attached to a `gnet` connection without worrying about data races.

Besides this new feature, v2.10.0 also ships with a bunch of enhancements and bug fixes, including using `net/url` for validating network addresses, ensuring listeners only open once via `sync.Once`, fixing `Conn.Write`/`Conn.Writev` to return non-retryable errors properly, setting the `FD_CLOEXEC` flag correctly when enrolling/duplicating client file descriptors, and fixing Unix domain socket address parsing for absolute paths.

For more details, please refer to the [release notes](https://github.com/panjf2000/gnet/releases/tag/v2.10.0).

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!