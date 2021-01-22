---
last_modified_on: "2020-09-13"
$schema: "/.meta/.schemas/highlights.json"
title: "Support locking OS thread"
description: "Support locking each I/O event-loop goroutine to an OS thread"
author_github: "https://github.com/panjf2000"
pr_numbers: [6fd6413]
release: "features"
hide_on_release_notes: false
tags: ["type: new feature", "domain: thread", "platform: arm64"]
---

## About this change

Add functional option `LockOSThread` used to determine whether each I/O event-loop is associated to an OS thread, it is useful when you need some kind of mechanisms like thread local storage, 
or invoke certain C libraries (such as graphics lib: GLib) that require thread-level manipulation via cgo, or want all I/O event-loops to actually run in parallel for a potential higher performance. 
