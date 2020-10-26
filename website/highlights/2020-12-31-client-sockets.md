---
last_modified_on: "2020-12-08"
$schema: "/.meta/.schemas/highlights.json"
title: "New feature allows creating eventloop managed client sockets"
description: "Open a client socket and react to events like accepted sockets"
author_github: "https://github.com/hellertime"
pr_numbers: [22b190f]
release: "features"
hide_on_release_notes: false
tags: ["type: new feature", "domain: client", "platform: unix"]
---

## About this change

Open connections to remote clients, and manage them as you do connections made to the Server. New Connection are created with a `Dial` method,
which is part of the `Conn` interface, this assures the new socket is managed by the same eventloop as the `Conn` that created it.
