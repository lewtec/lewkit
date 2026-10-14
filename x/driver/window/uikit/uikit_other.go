//go:build !darwin

package uikit

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func open(context.Context, window.Config) (window.Window, error) {
	return nil, driver.ForGOOS("ios")
}
