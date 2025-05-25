---
last_modified_on: "2025-05-25"
id: announcing-gnet-v2-8-0
title: Announcing gnet v2.8.0
description: "Hello World! We present you, gnet v2.8.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

![](/img/gnet-v2-8-0.jpg)

The `gnet` v2.8.0 is officially released!

***The most significant enhancement in this release is the support for registering new connections to event loops!***

Starting with v2.8.0, you're able to register new connections to an event loop, enabling you to delegate sockets that are not created by the `gnet` internally to the event loops of `gnet`.

This feature is particularly useful for proxy scenarios where you need to create and manage a mass of connections to backend servers and want to take advantage of the `gnet` event loops by registering these connections to the event loops. Once you a connection is registered to a event loop, you will be able to use the `gnet` APIs to operate the connection in a event-driven way, such as sending and receiving data, closing the connection, etc., as you do with the connections created by `gnet` itself.

The new feature is implemented by introducing a new `EventLoop` interface, which provides a set of methods for manipulating the event loops, including registering new connections and scheduling executions on the event loops, etc. This allows you to interact with the event loops in a more flexible and powerful way.

```go
// EventLoop provides a set of methods for manipulating the event-loop.
type EventLoop interface {
	// ============= Concurrency-safe methods =============

	// Register connects to the given address and registers the connection to the current event-loop.
	Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error)
	// Enroll is like Register, but it accepts an established net.Conn instead of a net.Addr.
	Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error)
	// Execute executes the given runnable on the event-loop at some time in the future.
	Execute(ctx context.Context, runnable Runnable) error
	// Schedule is like Execute, but it allows you to specify when the runnable is executed.
	// In other words, the runnable will be executed when the delay duration is reached.
	// TODO(panjf2000): not supported yet, implement this.
	Schedule(ctx context.Context, runnable Runnable, delay time.Duration) error

	// ============ Concurrency-unsafe methods ============

	// Close closes the given Conn that belongs to the current event-loop.
	// It must be called on the same event-loop that the connection belongs to.
	Close(Conn) error
}
```

Check out [gnet.EventLoop](https://pkg.go.dev/github.com/panjf2000/gnet/v2@v2.8.0#EventLoop) for more docs.

For more details, please refer to the [release notes](https://github.com/panjf2000/gnet/releases/tag/v2.8.0).

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!