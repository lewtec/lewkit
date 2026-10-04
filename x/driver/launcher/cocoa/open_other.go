//go:build !darwin

package cocoa

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func (backend) Choose(context.Context, launcher.ChooseOptions) (*launcher.Item, error) {
	return nil, fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}

func (backend) Prompt(context.Context, string) (string, error) {
	return "", fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}

func (backend) Confirm(context.Context, string) (bool, error) {
	return false, fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
}
