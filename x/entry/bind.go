package entry

import (
	"context"
	"errors"
)

var (
	ErrRunBoundEntryNoApp = errors.New("entry: no app")
)

var bound func(ctx context.Context) error

// Bind registers the process function the Android host calls after it loads
// the app library. Desktop processes keep using [Main].
func Bind(fn func(ctx context.Context) error) {
	bound = fn
}

// RunBound runs the function registered with [Bind].
// ctx is the process root. A nil context is an error.
func RunBound(ctx context.Context) error {
	if bound == nil {
		return ErrRunBoundEntryNoApp
	}
	if ctx == nil {
		return ErrNilContext
	}
	return bound(ctx)
}
