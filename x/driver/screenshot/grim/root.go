// Package grim captures Wayland screens with grim and slurp.
package grim

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/screenshot"
)

type factory struct{}

func (factory) ID() string   { return "screenshot_grim" }
func (factory) Name() string { return "Grim (Wayland)" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	return requireBinary("grim")
}

func (factory) New(context.Context) (screenshot.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[screenshot.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[screenshot.Driver](factory{})
}
