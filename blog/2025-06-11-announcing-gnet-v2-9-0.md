---
last_modified_on: "2025-06-11"
id: announcing-gnet-v2-9-0
title: Announcing gnet v2.9.0
description: "Hello World! We present you, gnet v2.9.0!"
author_github: https://github.com/panjf2000
tags: ["type: announcement", "domain: presentation"]
---

![](/img/gnet-v2-9-0.jpg)

The `gnet` v2.9.0 is officially released!

There are two new features come with this release:

- Support customizing `TCP_KEEPINTVL` and `TCP_KEEPCNT`
- Support running client with multiple event loops

In this release, we have added the ability to customize all three TCP keepalive options: `TCP_KEEPIDLE`, `TCP_KEEPINTVL`, and `TCP_KEEPCNT`, which allows developers to fine-tune the TCP keepalive settings for their applications and help accommodate various use cases of long-lived connections.

Additionally, we have introduced the ability to run a client with multiple event loops. It allows developers to take advantage of multi-core systems by distributing the workload across multiple event loops when working with the gnet client, which can help improve the performance and scalability of their applications.

For more details, please refer to the [release notes](https://github.com/panjf2000/gnet/releases/tag/v2.9.0).

P.S. Follow me on Twitter [@panjf2000](https://twitter.com/panjf2000) to get the latest updates about gnet!