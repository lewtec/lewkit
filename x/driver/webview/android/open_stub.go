//go:build !android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func (pageDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, driver.ForGOOS("android")
}
