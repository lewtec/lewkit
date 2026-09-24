package webview

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/daynight"
)

// Follow calls apply with the current day or night mode and each later change.
// apply runs on the Watch goroutine. A missing daynight driver is ignored.
func Follow(ctx context.Context, apply func(daynight.Mode)) {
	if apply == nil {
		return
	}
	changes, err := daynight.Watch(ctx)
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
