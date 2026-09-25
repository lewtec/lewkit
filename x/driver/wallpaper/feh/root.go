// Package feh sets an X11 wallpaper with feh.
package feh

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type factory struct{}

func (factory) ID() string   { return "wallpaper_feh" }
func (factory) Name() string { return "X11 (feh)" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "DISPLAY"); err != nil {
		return err
	}
	if err := requireBinary("systemd-run"); err != nil {
		return err
	}
	return requireBinary("feh")
}

func (factory) New(context.Context) (wallpaper.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[wallpaper.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[wallpaper.Driver](factory{})
}
