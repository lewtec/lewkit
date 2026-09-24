package webview

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/appearance"
)

// Follow calls apply with the current system scheme and each later change.
// apply runs on the Watch goroutine. A missing appearance driver is ignored.
func Follow(ctx context.Context, apply func(appearance.Scheme)) {
	if apply == nil {
		return
	}
	changes, err := appearance.Watch(ctx)
	if err != nil {
		return
	}
	go func() {
		for scheme := range changes {
			if ctx.Err() != nil {
				return
			}
			apply(scheme)
		}
	}()
}
