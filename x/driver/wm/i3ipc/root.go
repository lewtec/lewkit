// Package i3ipc drives Sway and i3 through swaymsg and i3-msg.
package i3ipc

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type swayFactory struct{}

func (swayFactory) ID() string   { return "wm_sway" }
func (swayFactory) Name() string { return "Sway" }
func (swayFactory) Weight() int  { return 60 }

func (swayFactory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	return requireBinary("swaymsg")
}

func (swayFactory) New(context.Context) (wm.Driver, error) {
	return backend{bin: "swaymsg"}, nil
}

type i3Factory struct{}

func (i3Factory) ID() string   { return "wm_i3" }
func (i3Factory) Name() string { return "i3" }
func (i3Factory) Weight() int  { return 40 }

func (i3Factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "DISPLAY"); err != nil {
		return err
	}
	return requireBinary("i3-msg")
}

func (i3Factory) New(context.Context) (wm.Driver, error) {
	return backend{bin: "i3-msg"}, nil
}

var _ driver.DriverFactory[wm.Driver] = swayFactory{}
var _ driver.Weighter = swayFactory{}
var _ driver.DriverFactory[wm.Driver] = i3Factory{}
var _ driver.Weighter = i3Factory{}

func init() {
	driver.Register[wm.Driver](swayFactory{})
	driver.Register[wm.Driver](i3Factory{})
}
