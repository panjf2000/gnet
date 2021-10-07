// Copyright (c) 2021 Andy Pan
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

//go:build freebsd || dragonfly || darwin
// +build freebsd dragonfly darwin

package io

import (
	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/errors"
)

// Writev simply calls write() multiple times cuz writev() on BSD-like OS's is not yet implemented in Go.
func Writev(fd int, iov [][]byte) (int, error) {
	var sum int
	for i := range iov {
		n, err := unix.Write(fd, iov[i])
		if err != nil {
			if sum == 0 {
				sum = n
			}
			return sum, err
		}
		sum += n
		if n < len(iov[i]) {
			return sum, errors.ErrShortWritev
		}
	}
	return sum, nil
}

// Readv simply calls read() multiple times cuz readv() on BSD-like OS's is not yet implemented in Go.
func Readv(fd int, iov [][]byte) (int, error) {
	var sum int
	for i := range iov {
		n, err := unix.Read(fd, iov[i])
		if err != nil {
			if sum == 0 {
				sum = n
			}
			return sum, err
		}
		sum += n
		if n < len(iov[i]) {
			return sum, errors.ErrShortReadv
		}
	}
	return sum, nil
}
