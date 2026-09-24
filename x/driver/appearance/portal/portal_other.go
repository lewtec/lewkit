//go:build !linux

package portal

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/appearance"
)

func available(context.Context) error {
	return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}

func open(context.Context) (appearance.Driver, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
