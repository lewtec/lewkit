// Package brightnessctl sets the backlight with brightnessctl.
package brightnessctl

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/brightness"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type factory struct{}

func (factory) ID() string   { return "brightness_brightnessctl" }
func (factory) Name() string { return "brightnessctl" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return requireBinary(ctx, "brightnessctl")
}

func (factory) New(context.Context) (brightness.Driver, error) { return backend{}, nil }

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

func init() {
	driver.Register[brightness.Driver](factory{})
}
