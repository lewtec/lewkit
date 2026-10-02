// Package x11 controls outputs through xset and xrandr.
package x11

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/screen"
)

type factory struct{}

func (factory) ID() string   { return "screen_x11" }
func (factory) Name() string { return "X11" }

// Weight is below Sway. XWayland often sets DISPLAY beside WAYLAND_DISPLAY.
func (factory) Weight() int { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "DISPLAY"); err != nil {
		return err
	}
	if err := execdriver.RequireBinary(ctx, "xset"); err != nil {
		return err
	}
	return execdriver.RequireBinary(ctx, "xrandr")
}

func (factory) New(context.Context) (screen.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[screen.Driver](factory{})
}
