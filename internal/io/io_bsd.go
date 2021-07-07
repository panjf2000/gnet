// Copyright (c) 2021 Andy Pan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
