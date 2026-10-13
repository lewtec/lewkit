// Package pace sleeps inside a taskgroup demo until the context ends.
package pace

import (
	"context"
	"time"
)

// Sleep waits for d or returns the context cause.
func Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}
