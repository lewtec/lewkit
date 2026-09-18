//go:build !windows

package win32

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (wdriver) Open(context.Context, window.Config) (window.Window, error) {
	return nil, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
