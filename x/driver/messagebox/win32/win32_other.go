//go:build !windows

package win32

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

func available(context.Context) error {
	return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}

func open(context.Context) (messagebox.Driver, error) {
	return nil, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
