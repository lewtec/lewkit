//go:build !windows

package win32

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type opener struct{}

func (opener) Open(context.Context, tray.Config) (tray.Tray, error) {
	return nil, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
