//go:build !darwin

package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (cdriver) Open(context.Context, window.Config) (window.Window, error) {
	return nil, driver.RequireGOOS("darwin")
}
