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

//go:build dragonfly || freebsd || netbsd || openbsd
// +build dragonfly freebsd netbsd openbsd

package socket

import errorx "github.com/panjf2000/gnet/v2/pkg/errors"

// SetBindToDevice is not implemented on *BSD because there is
// no equivalent of Linux's SO_BINDTODEVICE.
func SetBindToDevice(_ int, _ string) error {
	return errorx.ErrUnsupportedOp
}
