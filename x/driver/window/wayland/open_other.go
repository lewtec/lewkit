//go:build !linux || android

package wayland

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}

func (opener) Open(context.Context, window.Config) (window.Window, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
