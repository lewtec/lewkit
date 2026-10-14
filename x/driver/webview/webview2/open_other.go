//go:build !windows

package webview2

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func loader() error {
	return driver.ForGOOS("windows")
}

func (edgeDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, driver.ForGOOS("windows")
}
