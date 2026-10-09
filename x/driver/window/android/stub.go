//go:build !android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func init() { driver.Register[window.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "window_android" }
func (factory) Name() string { return "Android surface" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (factory) New(context.Context) (window.Driver, error) {
	return nil, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
