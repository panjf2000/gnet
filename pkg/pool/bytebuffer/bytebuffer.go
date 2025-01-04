// Copyright (c) 2019 The Gnet Authors. All rights reserved.
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

// Package bytebuffer is a pool of bytebufferpool.ByteBuffer.
package bytebuffer

import "github.com/valyala/bytebufferpool"

// ByteBuffer is the alias of bytebufferpool.ByteBuffer.
type ByteBuffer = bytebufferpool.ByteBuffer

var (
	// Get returns an empty byte buffer from the pool, exported from gnet/bytebuffer.
	Get = bytebufferpool.Get
	// Put returns byte buffer to the pool, exported from gnet/bytebuffer.
	Put = func(b *ByteBuffer) {
		if b != nil {
			bytebufferpool.Put(b)
		}
	}
)
