---
last_modified_on: "2024-11-09"
$schema: "/.meta/.schemas/highlights.json"
title: "Release of gnet v2.6.0"
description: "Release of the official stable version of v2.6.0"
author_github: "https://github.com/panjf2000"
pr_numbers: [2b0357c]
release: "2.6.0"
hide_on_release_notes: false
tags: ["type: tag", "domain: v2.6.0"]
---

![](/img/gnet-v2-6-0.webp)

I hereby announce the release of [gnet v2.6.0](https://github.com/panjf2000/gnet/releases/tag/v2.6.0), where we've added two major features of [`SO_BINDTODEVICE`](https://man7.org/linux/man-pages/man7/socket.7.html) and [configurable edge-triggered chunk](https://github.com/panjf2000/gnet/pull/646), along with a few bug-fixes.

Another change to note is that starting with this release, the minimum required Go version to run `gnet` is 1.20!
