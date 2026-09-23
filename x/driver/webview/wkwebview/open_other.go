//go:build !darwin

package wkwebview

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func frameworks() error {
	return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}

func (webKitDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}
