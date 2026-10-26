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

//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && poll_opt

package netpoll

import "unsafe"

// The kernel retains the address after this call. Keep the caller's attachment
// on the heap even though its representation is copied without pointer stores.
//
//go:uintptrescapes
func convertPollAttachment(ptr unsafe.Pointer, attachment uintptr) {
	// epoll_data is packed at an address that need not be pointer-aligned.
	// Copy its representation as bytes rather than dereferencing a **T there.
	*(*[unsafe.Sizeof(attachment)]byte)(ptr) = *(*[unsafe.Sizeof(attachment)]byte)(unsafe.Pointer(&attachment))
}

func restorePollAttachment(ptr unsafe.Pointer) *PollAttachment {
	var attachment *PollAttachment
	*(*[unsafe.Sizeof(attachment)]byte)(unsafe.Pointer(&attachment)) = *(*[unsafe.Sizeof(attachment)]byte)(ptr)
	return attachment
}
