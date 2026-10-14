//go:build !darwin

package wkwebview

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func frameworks() error {
	return driver.ForGOOS("darwin")
}

func (webKitDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, driver.ForGOOS("darwin")
}
