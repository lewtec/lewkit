package entry

import (
	"context"
	"fmt"
)

var bound func(ctx context.Context) error

// Bind registers the process function the Android host calls after it loads
// the app library. Desktop processes keep using [Main].
func Bind(fn func(ctx context.Context) error) {
	bound = fn
}

// RunBound runs the function registered with [Bind].
func RunBound() error {
	if bound == nil {
		return fmt.Errorf("entry: no app")
	}
	return bound(context.Background())
}
