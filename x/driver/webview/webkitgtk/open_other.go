//go:build !linux

package webkitgtk

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func libraries(context.Context) error {
	return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}

func (gtkDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
