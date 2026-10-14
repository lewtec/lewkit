//go:build !linux || android

package wayland

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS("linux")
}

func (opener) Open(context.Context, window.Config) (window.Window, error) {
	return nil, driver.ForGOOS("linux")
}
