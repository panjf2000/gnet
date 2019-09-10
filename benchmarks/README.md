## evio benchmark tools

Required tools:

- [bombardier](https://github.com/codesenberg/bombardier) for HTTP
- [tcpkali](https://github.com/machinezone/tcpkali) for Echo
- [Redis](http://redis.io) for Redis

Required Go packages:

```
go get gonum.org/v1/plot/...
go get -u github.com/valyala/fasthttp
go get -u github.com/tidwall/redcon
```

And of course [Go](https://golang.org) is required.

Run `bench.sh` for all benchmarks.

## Notes

- The current results were run on an Ec2 c4.xlarge instance.
- The servers started in single-threaded mode (GOMAXPROC=1).
- Network clients connected over Ipv4 localhost.

Like all benchmarks ever made in the history of whatever, YMMV. Please tweak and run in your environment and let me know if you see any glaring issues.
