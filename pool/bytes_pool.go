// Copyright 2019 Andy Pan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package pool

import "github.com/gobwas/pool/pbytes"

//const (
//	// Lower represents the lower value of the logarithmic range of the pbytes.Pool.
//	Lower = 64
//
//	// Upper represents the upper value of the logarithmic range of the pbytes.Pool.
//	Upper = 64 * 1024
//)

// BytesPool is the alias of pbytes.Pool.
type BytesPool = pbytes.Pool

func PutBytes(buf []byte) {
	pbytes.Put(buf)
}

// NewBytesPool instantiates a *BytesPool that reuses slices which size is in logarithmic range [Lower, Upper].
func NewBytesPool() *BytesPool {
	//return pbytes.New(Lower, Upper)
	return pbytes.DefaultPool
}
