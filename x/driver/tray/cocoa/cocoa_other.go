//go:build !darwin

package cocoa

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type opener struct{}

func (opener) Open(context.Context, tray.Config) (tray.Tray, error) {
	return nil, fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}
