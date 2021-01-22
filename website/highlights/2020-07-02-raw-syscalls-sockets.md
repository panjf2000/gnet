---
last_modified_on: "2020-07-02"
$schema: "/.meta/.schemas/highlights.json"
title: "New approach to create server sockets"
description: "Leverage system calls instead of `net` package to create sockets"
author_github: "https://github.com/panjf2000"
pr_numbers: [ccc8c64]
release: "features"
hide_on_release_notes: false
tags: ["type: optimization", "domain: socket", "platform: arm64"]
---

## About this change

Benefit from system calls provided by Go standard library, server listener sockets inside gnet have gotten rid of the Go `net` package eventually,
there are no more methods from `net` package involved in the process of creating the listener of server socket, all by raw system calls. 
