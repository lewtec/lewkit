// Package swaybg sets a Wayland wallpaper with swaybg.
package swaybg

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
)

type factory struct{}

func (factory) ID() string   { return "wallpaper_swaybg" }
func (factory) Name() string { return "Wayland (swaybg)" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	if err := requireBinary("systemd-run"); err != nil {
		return err
	}
	return requireBinary("swaybg")
}

func (factory) New(context.Context) (wallpaper.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[wallpaper.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[wallpaper.Driver](factory{})
}
