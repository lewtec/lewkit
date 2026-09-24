//go:build !darwin

package cocoa

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/colorscheme"
)

func available(context.Context) error {
	return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}

func open(context.Context) (colorscheme.Driver, error) {
	return nil, fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}
