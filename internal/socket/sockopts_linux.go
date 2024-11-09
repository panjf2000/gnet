// Copyright (c) 2024 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package socket

import (
	"os"

	"golang.org/x/sys/unix"
)

// SetBindToDevice binds the socket to a specific network interface.
//
// SO_BINDTODEVICE on Linux works in both directions: only process packets
// received from the particular interface along with sending them through
// that interface, instead of following the default route.
func SetBindToDevice(fd int, ifname string) error {
	return os.NewSyscallError("setsockopt", unix.BindToDevice(fd, ifname))
}
