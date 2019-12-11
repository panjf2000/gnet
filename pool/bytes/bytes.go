// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package bytes

import "github.com/gobwas/pool/pbytes"

//const (
//	// Lower represents the lower value of the logarithmic range of the pbytes.Pool.
//	Lower = 64
//
//	// Upper represents the upper value of the logarithmic range of the pbytes.Pool.
//	Upper = 64 * 1024
//)

// Pool is the alias of pbytes.Pool.
type Pool = pbytes.Pool

// Put puts bytes back to pool.
func Put(buf []byte) {
	pbytes.Put(buf)
}

// Default instantiates a *BytesPool that reuses slices which size is in logarithmic range [Lower, Upper].
func Default() *Pool {
	//return pbytes.Default(Lower, Upper)
	return pbytes.DefaultPool
}
