## gnet benchmark tools

Required tools:

- [bombardier](https://github.com/codesenberg/bombardier) for HTTP
- [tcpkali](https://github.com/machinezone/tcpkali) for Echo

Required Go packages:

```
go get gonum.org/v1/plot/...
go get -u github.com/valyala/fasthttp
```

And of course [Go](https://golang.org) is required.

Run `bench.sh` for all benchmarks.

## Notes

- The current results were run on an c5.2xlarge instance (8 Virtual CPUs, 16.0 GiB Memory, 120 GiB SSD (EBS) Storage).
- The servers started in multiple-threaded mode (GOMAXPROC=1).
- Network clients connected over Ipv4 localhost.

Like all benchmarks ever made in the history of whatever, YMMV. Please tweak and run in your environment and let me know if you see any glaring issues.

# Benchmark Test

## On Linux (epoll)

### Test Environment

```powershell
Go Version: go1.12.9 linux/amd64
Machine:    Amazon c5.2xlarge
OS:         Ubuntu 18.04
CPU:        8 Virtual CPUs
Memory:     16.0 GiB
```

### Echo Server

![](results/echo_linux.png)

### HTTP Server

![](results/http_linux.png)

## On FreeBSD (kqueue)

### Test Environment

```powershell
Go Version: go version go1.12.9 darwin/amd64
Machine:    MacBook Pro
OS:         macOS Mojave 10.14.6
CPU:        4 CPUs
Memory:     8.0 GiB
```

### Echo Server

![](results/echo_mac.png)

### HTTP Server

![](results/http_mac.png)

