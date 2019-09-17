# go-reuseport

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![](https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square)](https://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23libp2p-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23libp2p)
[![codecov](https://codecov.io/gh/libp2p/go-reuseport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-reuseport)
[![Travis CI](https://travis-ci.org/libp2p/go-reuseport.svg?branch=master)](https://travis-ci.org/libp2p/go-reuseport)

**NOTE:** This package REQUIRES go >= 1.11.

This package enables listening and dialing from _the same_ TCP or UDP port.
This means that the following sockopts may be set:

```
SO_REUSEADDR
SO_REUSEPORT
```

- godoc: https://godoc.org/github.com/libp2p/go-reuseport

This is a simple package to help with address reuse. This is particularly
important when attempting to do TCP NAT holepunching, which requires a process
to both Listen and Dial on the same TCP port. This package provides some
utilities around enabling this behaviour on various OS.

## Examples


```Go
// listen on the same port. oh yeah.
l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
l2, _ := reuse.Listen("tcp", "127.0.0.1:1234")
```

```Go
// dial from the same port. oh yeah.
l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
l2, _ := reuse.Listen("tcp", "127.0.0.1:1235")
c, _ := reuse.Dial("tcp", "127.0.0.1:1234", "127.0.0.1:1235")
```

**Note: cant dial self because tcp/ip stacks use 4-tuples to identify connections, and doing so would clash.**

## Tested

Tested on `darwin`, `linux`, and `windows`.
