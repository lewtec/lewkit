//go:build !linux || android

package webkitgtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func libraries(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return driver.ForGOOS("linux")
}

func (gtkDriver) Open(ctx context.Context, _ webview.Config) (webview.View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, driver.ForGOOS("linux")
}
