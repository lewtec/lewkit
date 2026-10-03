package uikit

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_uikit" }
func (factory) Name() string { return "UIKit" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (window.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	return open(ctx, cfg)
}
