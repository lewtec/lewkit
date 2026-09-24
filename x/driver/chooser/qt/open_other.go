//go:build !linux

package qt

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/chooser"
)

func (opener) Choose(context.Context, chooser.Request) ([]string, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
