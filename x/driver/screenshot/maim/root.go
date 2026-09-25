// Package maim captures X11 screens with maim and slop.
package maim

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/screenshot"
)

type factory struct{}

func (factory) ID() string   { return "screenshot_maim" }
func (factory) Name() string { return "Maim (X11)" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "DISPLAY"); err != nil {
		return err
	}
	return requireBinary("maim")
}

func (factory) New(context.Context) (screenshot.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[screenshot.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[screenshot.Driver](factory{})
}
