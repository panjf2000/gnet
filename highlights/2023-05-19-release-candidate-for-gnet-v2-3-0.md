---
last_modified_on: "2023-05-19"
$schema: "/.meta/.schemas/highlights.json"
title: "Release candidate for gnet v2.3.0"
description: "The first release candidate for gnet v2.3.0"
author_github: "https://github.com/panjf2000"
pr_numbers: [b493107]
release: "2.3.0-rc.1"
hide_on_release_notes: false
tags: ["type: tag", "domain: v2.3.0-rc.1"]
---

## Intro

The two major updates in this release candidate are [#460](https://github.com/panjf2000/gnet/pull/460) and [#461](https://github.com/panjf2000/gnet/pull/461).

We introduced a new data structure `matrix` in [#460](https://github.com/panjf2000/gnet/pull/460) to displace the default `map` for managing connections internally, with the help of this new data structure, we can significantly reduce GC (Garbage Collection) latency:

```bash
goos: darwin
goarch: arm64
pkg: github.com/panjf2000/gnet/v2
                                    │     old      │                 new                  │
                                    │    sec/op    │    sec/op     vs base                │
GC4El100k/Run-4-eventloop-100000-10    30.74m ± 3%   19.68m ± 10%  -35.98% (p=0.000 n=10)
GC4El200k/Run-4-eventloop-200000-10    63.64m ± 3%   38.16m ± 11%  -40.04% (p=0.000 n=10)
GC4El500k/Run-4-eventloop-500000-10   177.28m ± 8%   95.21m ±  4%  -46.29% (p=0.000 n=10)
geomean                                70.26m        41.51m        -40.92%

                                    │     old     │                new                 │
                                    │    B/op     │    B/op      vs base               │
GC4El100k/Run-4-eventloop-100000-10   27.50 ± 35%   25.50 ± 33%       ~ (p=0.423 n=10)
GC4El200k/Run-4-eventloop-200000-10   27.50 ± 53%   20.50 ± 66%       ~ (p=0.642 n=10)
GC4El500k/Run-4-eventloop-500000-10   16.00 ±   ?   18.00 ±   ?       ~ (p=0.357 n=10)
geomean                               22.96         21.11        -8.04%

                                    │     old      │                 new                 │
                                    │  allocs/op   │ allocs/op   vs base                 │
GC4El100k/Run-4-eventloop-100000-10   0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
GC4El200k/Run-4-eventloop-200000-10   0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
GC4El500k/Run-4-eventloop-500000-10   0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
geomean                                          ²               +0.00%                ²
¹ all samples are equal
² summaries must be >0 to compute geomean
```

**The more connections there are, the more pronounced the effect.**

While we have performed sufficient testing on `matrix`, we are still using `map` as the default connection storage in this RC version for the sake of caution, but you can enable the new data structure by specifying build tags: -tags=gc_opt. This can be considered as a precautionary measure so that in case `matrix` has any unexpected bugs, you can quickly fall back to the default `map`. We will consider promoting `matrix` to be the default storage for connections in a subsequent official release.

Another significant leap is [#461](https://github.com/panjf2000/gnet/pull/461), you can now run `gnet` on Windows, it should be noted that the Windows version of `gnet` is intended for development purposes and is not recommended for use in production.
