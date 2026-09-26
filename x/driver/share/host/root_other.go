//go:build !android && !ios

package host

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: not android or ios", driver.ErrIncompatible)
}
