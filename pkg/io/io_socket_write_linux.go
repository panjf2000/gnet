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

//go:build linux && !gnet_sendto

package io

// WriteSocketNB preserves the existing write(2) path unless the gnet_sendto
// build tag is enabled. Keeping this switch at build time lets the benchmark
// compare the send syscall independently without a runtime branch.
func WriteSocketNB(fd int, p []byte) (int, error) { return WriteNB(fd, p) }
