---
last_modified_on: "2023-06-21"
id: benchmark
title: "Benchmark"
description: "Benchmark results with other similar frameworks in Go"
---

## Benchmarks on TechEmpower

```bash
# Hardware Environment
* 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
* 32GB RAM
* Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
* Dedicated Cisco 10-gigabit Ethernet switch
* Go1.19.x linux/amd64
```

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-top50-light.jpg)

This is a leaderboard of the top ***50*** out of ***499*** frameworks that encompass various programming languages worldwide, in which `gnet` is ranked ***first***.

![](https://raw.githubusercontent.com/panjf2000/illustrations/master/benchmark/techempower-plaintext-topN-go-light.png)

This is the full framework ranking of Go and `gnet` tops all the other frameworks, which makes `gnet` the ***fastest*** networking framework in Go.

To see the full ranking list, visit [TechEmpower Plaintext Benchmark](https://www.techempower.com/benchmarks/#section=test&runid=a07a7117-f861-49b2-a710-94970c5767d0&test=plaintext).

## Contrasts to the similar networking libraries

## On Linux (epoll)

### Test Environment

```bash
# Machine information
        OS : Ubuntu 20.04/x86_64
       CPU : 8 CPU cores, AMD EPYC 7K62 48-Core Processor
    Memory : 16.0 GiB

# Go version and settings
Go Version : go1.17.2 linux/amd64
GOMAXPROCS : 8

# Benchmark parameters
TCP connections : 1000/2000/5000/10000
Packet size     : 512/1024/2048/4096/8192/16384/32768/65536 bytes
Test duration   : 15s
```

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_linux.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_linux.png)

## On MacOS (kqueue)

### Test Environment

```bash
# Machine information
        OS : MacOS Big Sur/x86_64
       CPU : 6 CPU cores, Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
    Memory : 16.0 GiB

# Go version and settings
Go Version : go1.16.5 darwin/amd64
GOMAXPROCS : 12

# Benchmark parameters
TCP connections : 300/400/500/600/700
Packet size     : 512/1024/2048/4096/8192 bytes
Test duration   : 15s
```

#### [Echo benchmark](https://github.com/gnet-io/gnet-benchmarks)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_conn_macos.png)

![](https://github.com/panjf2000/gnet_benchmarks/raw/master/results/echo_packet_macos.png)