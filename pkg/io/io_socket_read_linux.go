// Copyright (c) 2025 The Gnet Authors. All rights reserved.
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

//go:build linux && gnet_legacy_read

package io

// ReadSocketNB preserves the legacy read(2) path when gnet_legacy_read is set.
// The default Linux build uses recvfrom(2) for socket descriptors.
func ReadSocketNB(fd int, p []byte) (int, error) { return ReadNB(fd, p) }
