//go:build !android || !cgo

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func init() { driver.Register[launcher.Confirmer](factory{}) }

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (factory) New(context.Context) (launcher.Confirmer, error) {
	return nil, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
