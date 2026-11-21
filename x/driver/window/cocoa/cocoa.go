package cocoa

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_cocoa" }
func (factory) Name() string { return "Cocoa" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (window.Driver, error) {
	return cdriver{}, nil
}

type cdriver struct{}
