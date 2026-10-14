// Package sway controls outputs through swaymsg.
package sway

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/screen"
)

type factory struct{}

func (factory) ID() string   { return "screen_sway" }
func (factory) Name() string { return "Sway" }

// Weight is above X11 so a Wayland session wins when DISPLAY is also set.
func (factory) Weight() int { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireEnvBinary(ctx, "WAYLAND_DISPLAY", "swaymsg")
}

func (factory) New(context.Context) (screen.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[screen.Driver](factory{})
}
