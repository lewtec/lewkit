// Package hyprland drives Hyprland through hyprctl.
package hyprland

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type factory struct{}

func (factory) ID() string   { return "wm_hyprland" }
func (factory) Name() string { return "Hyprland" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "HYPRLAND_INSTANCE_SIGNATURE"); err != nil {
		return err
	}
	return requireBinary("hyprctl")
}

func (factory) New(context.Context) (wm.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[wm.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[wm.Driver](factory{})
}
