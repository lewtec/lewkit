//go:build !android || !cgo

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func open(context.Context) (backend, error) {
	return backend{}, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

type backend struct{}

func (backend) Choose(context.Context, launcher.ChooseOptions) (*launcher.Item, error) {
	return nil, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (backend) Prompt(context.Context, string) (string, error) {
	return "", fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (backend) Confirm(context.Context, string) (bool, error) {
	return false, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
