//go:build !linux

package qt

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

func (opener) Choose(context.Context, filedialog.Request) ([]string, error) {
	return nil, fmt.Errorf("%w: not linux", driver.ErrIncompatible)
}
