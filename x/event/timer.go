package event

import (
	"context"
	"time"
)

// CreateTimer sends every d until ctx is done. The channel holds one tick;
// a slow consumer drops extras instead of queuing catch-up. The channel is
// not closed; stop by cancelling ctx.
func CreateTimer(ctx context.Context, d time.Duration) <-chan struct{} {
	ch := make(chan struct{}, 1)
	if d <= 0 {
		d = time.Nanosecond
	}
	go func() {
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()
	return ch
}
