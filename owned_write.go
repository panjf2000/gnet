// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.

package gnet

import (
	"context"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
)

// OwnedWriter optionally borrows output without copying it into a pooled ring.
// Call WriteOwned only on the connection's event loop. Keep the entire backing
// allocation immutable and retained until release runs. release runs exactly
// once, including rejection; it may run before WriteOwned returns. A nil error
// means every byte reached the socket; an error means the write was abandoned.
// Release callbacks run after buffer unlinking and may reenter the connection.
type OwnedWriter interface {
	WriteOwned(data []byte, release func(error)) (int, error)
}

// ExecuteHighPriority uses the same FIFO queue as native AsyncWrite. Like
// EventLoop.Execute, accepted work can be abandoned when its owner exits.
// Callers must retire that work after observing the owner's completed shutdown.
// Unsupported EventLoop wrappers are rejected; there is no low-priority fallback.
//
//nolint:revive // Preserve the owner-first signature of the proposed downstream API.
func ExecuteHighPriority(owner EventLoop, ctx context.Context, run Runnable) error {
	executor, ok := owner.(interface {
		ExecuteHighPriority(context.Context, Runnable) error
	})
	if !ok {
		return errorx.ErrUnsupportedOp
	}
	return executor.ExecuteHighPriority(ctx, run)
}
