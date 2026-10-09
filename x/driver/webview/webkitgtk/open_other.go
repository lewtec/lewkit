//go:build !linux || android

package webkitgtk

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func libraries(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}

func (gtkDriver) Open(ctx context.Context, _ webview.Config) (webview.View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
