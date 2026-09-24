//go:build !linux

package portal

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

// Available reports that the portal file chooser is a Linux driver.
func Available(context.Context, string) error {
	return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}

// Choose reports that the portal file chooser is a Linux driver.
func Choose(context.Context, string, filedialog.Request) ([]string, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
