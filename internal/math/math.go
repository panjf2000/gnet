// Copyright (c) 2022 The Gnet Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package math

const (
	bitSize       = 32 << (^uint(0) >> 63)
	maxintHeadBit = 1 << (bitSize - 2)
)

// IsPowerOfTwo reports whether the given n is a power of two.
func IsPowerOfTwo(n int) bool {
	return n > 0 && n&(n-1) == 0
}

// CeilToPowerOfTwo returns n if it is a power-of-two, otherwise the next-highest power-of-two.
func CeilToPowerOfTwo(n int) int {
	if n&maxintHeadBit != 0 && n > maxintHeadBit {
		panic("argument is too large")
	}

	if n <= 2 {
		return 2
	}

	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	return n
}

// FloorToPowerOfTwo returns n if it is a power-of-two, otherwise the next-highest power-of-two.
func FloorToPowerOfTwo(n int) int {
	if n <= 2 {
		return n
	}

	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16

	return n - (n >> 1)
}

// ClosestPowerOfTwo returns n if it is a power-of-two, otherwise the closest power-of-two.
func ClosestPowerOfTwo(n int) int {
	next := CeilToPowerOfTwo(n)
	if prev := next / 2; (n - prev) < (next - n) {
		next = prev
	}
	return next
}
