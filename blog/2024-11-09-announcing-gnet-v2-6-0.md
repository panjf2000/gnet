---
last_modified_on: "2024-11-09"
id: announcing-gnet-v2-6-0
title: Announcing gnet v2.6.0
description: "Hello World! We present you, gnet v2.6.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

![](/img/gnet-v2-6-0.webp)

The `gnet` v2.6.0 is officially released!

The two major updates in this release are [feat: support configurable I/O chunk to drain at a time in edge-triggered mode](https://github.com/panjf2000/gnet/pull/646) and [feat: support SO_BINDTODEVICE on Linux](https://github.com/panjf2000/gnet/pull/650).

## Intro

In `gnet` v2.5.0, we implemented edge-triggered I/O where the default chunk to read/write per event loop is 1MB and it's a constant value. However, in some scenarios, developers may want to adjust the chunk size to optimize the performance of their applications. In this release, we've added a new feature that allows developers to configure the I/O chunk size to drain/pump at a time in edge-triggered mode. This feature is especially useful for applications that require fine-tuning the I/O chunk size to achieve better performance.

Along with this new release comes another highlight: the support for `SO_BINDTODEVICE` on Linux. This feature allows developers to bind a `gnet` server to a specific network interface on a multi-homed host, which enables them to receive unicast packets only from that particular network interface and ignore packets from other interfaces. It is particularly useful for UDP servers that only want to process unicast packets from one interface while receiving broadcast packets.

Moreover, some critical bug fixes are included in this release, which further enhances the stability and reliability of `gnet`. Thus, we recommend all users to upgrade to the latest version to enjoy the new features and improvements. It should be noted that starting from this release, `gnet` will only support Go 1.20 or later versions.

For more details, please refer to the [release notes](https://github.com/panjf2000/gnet/releases/tag/v2.6.0).

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!