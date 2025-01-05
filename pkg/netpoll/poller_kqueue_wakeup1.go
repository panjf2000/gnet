//  Copyright (c) 2024 The Gnet Authors. All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

//go:build netbsd || openbsd

package netpoll

import (
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// TODO(panjf2000): NetBSD didn't implement EVFILT_USER for user-established events
// until NetBSD 10.0, check out https://www.netbsd.org/releases/formal-10/NetBSD-10.0.html
// Therefore we use the pipe to wake up the kevent on NetBSD at this point. Get back here
// and switch to EVFILT_USER when we bump up the minimal requirement of NetBSD to 10.0.
// Alternatively, maybe we can use EVFILT_USER on the NetBSD by checking the kernel version
// via uname(3) and fall back to the pipe if the kernel version is older than 10.0.

func (p *Poller) addWakeupEvent() error {
	p.pipe = make([]int, 2)
	if err := unix.Pipe2(p.pipe[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		logging.Fatalf("failed to create pipe for wakeup event: %v", err)
	}
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  uint64(p.pipe[0]),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_ADD,
	}}, nil, nil)
	return err
}

func (p *Poller) wakePoller() error {
retry:
	_, err := unix.Write(p.pipe[1], []byte("x"))
	if err == nil || err == unix.EAGAIN {
		return nil
	}
	if err == unix.EINTR {
		goto retry
	}
	logging.Warnf("failed to write to the wakeup pipe: %v", err)
	return err
}

func (p *Poller) drainWakeupEvent() {
	var buf [8]byte
	_, _ = unix.Read(p.pipe[0], buf[:])
}
