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

//go:build darwin || dragonfly || freebsd

package netpoll

import (
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/pkg/logging"
)

func (p *Poller) addWakeupEvent() error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil)
	return err
}

func (p *Poller) wakePoller() error {
retry:
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	if err == nil {
		return nil
	}
	if err == unix.EINTR {
		// All changes contained in the changelist should have been applied
		// before returning EINTR. But let's be skeptical and retry it anyway,
		// to make a 100% commitment.
		goto retry
	}
	logging.Warnf("failed to wake up the poller: %v", err)
	return err
}

func (p *Poller) drainWakeupEvent() {}
