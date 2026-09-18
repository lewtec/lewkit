package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_cocoa" }
func (factory) Name() string { return "Cocoa" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.RequireGOOS("darwin")
}

func (factory) New(context.Context) (window.Driver, error) {
	return cdriver{}, nil
}

type cdriver struct{}
