# gnet
High performance, lightweight, nonblocking network library written in pure Go, derived from [evio](https://github.com/tidwall/evio), but faster.

# Benchmark Test

## On Linux (epoll)

### Echo Server

![](benchmarks/results/echo_linux.png)

### HTTP Server

![](benchmarks/results/http_linux.png)

## On MacOS (kqueue)

### Echo Server

![](benchmarks/results/echo_mac.png)

### HTTP Server

![](benchmarks/results/http_mac.png)