//go:build !android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func (pageDriver) Open(context.Context, webview.Config) (webview.View, error) {
	return nil, fmt.Errorf("%w: not android", driver.ErrIncompatible)
}
