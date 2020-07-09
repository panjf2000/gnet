---
last_modified_on: "2019-11-07"
$schema: "/.meta/.schemas/highlights.json"
title: "Support multiple codecs"
description: "Support multiple codecs to encode/decode TCP stream"
author_github: "https://github.com/panjf2000"
pr_numbers: [2369df7]
release: "features"
hide_on_release_notes: false
tags: ["type: new feature", "domain: load-balancing"]
---

## About this change

In this change, gnet integrates multiple codecs to encode/decode network frames into/from TCP stream: LineBasedFrameCodec, DelimiterBasedFrameCodec, FixedLengthFrameCodec and LengthFieldBasedFrameCodec, referencing [netty codec](https://netty.io/4.1/api/io/netty/handler/codec/package-summary.html), 
in addition to these built-in codecs, gnet also supports customized codecs by functional option `Codec`.