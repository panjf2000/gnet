// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

//go:build linux && !poll_opt

package netpoll

func pollerWakeFD(poller *Poller) int { return poller.efd }
