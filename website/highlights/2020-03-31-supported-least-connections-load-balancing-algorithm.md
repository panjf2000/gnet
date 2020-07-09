---
last_modified_on: "2020-03-31"
$schema: "/.meta/.schemas/highlights.json"
title: "Support new load-balancing algorithm"
description: "Support new load-balancing algorithm of least-connections"
author_github: "https://github.com/panjf2000"
pr_numbers: [fc73013]
release: "features"
hide_on_release_notes: false
tags: ["type: new feature", "domain: load-balancing"]
---

## About this change

In the past, gnet has only one load-balancing algorithm: Round-Robin, now it supports a new one: Least-Connections
and provide the new functional option `LB`, enabling users to switch load-balancing algorithm at their own sweet will.

