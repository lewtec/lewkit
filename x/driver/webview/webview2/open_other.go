//go:build !windows

package webview2

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func loader() error {
	return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}

func (edgeDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
