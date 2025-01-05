---
last_modified_on: "2025-01-04"
id: announcing-gnet-v2-7-0
title: Announcing gnet v2.7.0
description: "Hello World! We present you, gnet v2.7.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

![](/img/gnet-v2-7-0.jpg)

The `gnet` v2.7.0 is officially released!

***In this release, most of the core internal packages used by gnet are now available outside of gnet!***

Take `netpoll` package as an example:

Package `netpoll` provides a portable event-driven interface for network I/O.

The underlying facility of event notification is OS-specific:
  - [epoll](https://man7.org/linux/man-pages/man7/epoll.7.html) on Linux
  - [kqueue](https://man.freebsd.org/cgi/man.cgi?kqueue) on *BSD/Darwin

With the help of the `netpoll` package, you can easily build your own high-performance
event-driven network applications based on epoll/kqueue.

The `Poller` represents the event notification facility whose backend is epoll or kqueue.
The `OpenPoller` function creates a new `Poller` instance:

```go
	poller, err := netpoll.OpenPoller()
	if err != nil {
		// handle error
	}

	defer poller.Close()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9090")
	if err != nil {
		// handle error
	}
	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		// handle error
	}

	f, err := c.File()
	if err != nil {
		// handle error
	}

	closeClient := func() {
		c.Close()
		f.Close()
	}
	defer closeClient()
```

The `PollAttachment` consists of a file descriptor and its callback function.
`PollAttachment` is used to register a file descriptor to `Poller`.
The callback function is called when an event occurs on the file descriptor:

```go
	pa := netpoll.PollAttachment{
		FD: int(f.Fd()),
		Callback: func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {
			if netpoll.IsErrorEvent(event, flags) {
				closeClient()
				return errors.ErrEngineShutdown
			}

			if netpoll.IsReadEvent(event) {
				buf := make([]byte, 64)
				// Read data from the connection.
				_, err := c.Read(buf)
				if err != nil {
					closeClient()
					return errors.ErrEngineShutdown
				}
				// Process the data...
			}

			if netpoll.IsWriteEvent(event) {
				// Write data to the connection.
				_, err := c.Write([]byte("hello"))
				if err != nil {
					closeClient()
					return errors.ErrEngineShutdown
				}
			}

			return nil
		}}

	if err := poller.AddReadWrite(&pa, false); err != nil {
		// handle error
	}
```

The `Poller.Polling` function starts the event loop monitoring file descriptors and
waiting for I/O events to occur:

```go
	poller.Polling(func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {
		return pa.Callback(fd, event, flags)
	})
```

Or

```go
	poller.Polling()
```

if you've enabled the build tag `poll_opt`.

Check out [gnet/pkg](https://pkg.go.dev/github.com/panjf2000/gnet/v2@v2.7.0/pkg) for more details.

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!