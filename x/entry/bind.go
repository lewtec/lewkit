package entry

import (
	"context"

	"github.com/lewtec/lewkit/x/entry/internal/androidcall"
)

var (
	ErrRunBoundEntryNoApp = androidcall.ErrRunEntryNoApp
)

var bound func(ctx context.Context) error

// Bind registers the process function the Android host calls after it loads
// the app library. Desktop processes keep using [Main].
func Bind(fn func(ctx context.Context) error) {
	bound = fn
}

// RunBound runs the function registered with [Bind].
// ctx is the process root.
func RunBound(ctx context.Context) error {
	if bound == nil {
		return ErrRunBoundEntryNoApp
	}
	return bound(ctx)
}
