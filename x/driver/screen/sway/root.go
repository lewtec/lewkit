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
	if err := driver.RequireEnv(ctx, "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	return requireBinary(ctx, "swaymsg")
}

func (factory) New(context.Context) (screen.Driver, error) { return backend{}, nil }

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

func init() {
	driver.Register[screen.Driver](factory{})
}
