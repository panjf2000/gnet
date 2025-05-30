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

import errorx "github.com/panjf2000/gnet/v2/pkg/errors"

// SetKeepAlivePeriod is not implemented on OpenBSD because there are
// no equivalents of Linux's TCP_KEEPIDLE, TCP_KEEPINTVL, and TCP_KEEPCNT.
func SetKeepAlivePeriod(_, _ int) error {
	// OpenBSD has no user-settable per-socket TCP keepalive options.
	return errorx.ErrUnsupportedOp
}

// SetKeepAlive is not implemented on OpenBSD.
func SetKeepAlive(_ int, _ bool, _, _, _ int) error {
	// OpenBSD has no user-settable per-socket TCP keepalive options.
	return errorx.ErrUnsupportedOp
}
