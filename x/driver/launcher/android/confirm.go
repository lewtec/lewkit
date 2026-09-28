package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

const (
	confirmNo         = 0
	confirmYes        = 1
	confirmNoActivity = 2
)

func decodeConfirm(code int) (bool, error) {
	switch code {
	case confirmYes:
		return true, nil
	case confirmNo:
		return false, nil
	case confirmNoActivity:
		return false, fmt.Errorf("%w: no foreground activity", driver.ErrUnavailable)
	default:
		return false, fmt.Errorf("%w: confirm code %d", driver.ErrUnavailable, code)
	}
}

func takeConfirm(ctx context.Context, ch <-chan int) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case code := <-ch:
		return decodeConfirm(code)
	}
}
