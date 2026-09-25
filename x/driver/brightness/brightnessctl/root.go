// Package brightnessctl sets the backlight with brightnessctl.
package brightnessctl

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/brightness"
)

type factory struct{}

func (factory) ID() string   { return "brightness_brightnessctl" }
func (factory) Name() string { return "brightnessctl" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return requireBinary("brightnessctl")
}

func (factory) New(context.Context) (brightness.Driver, error) { return backend{}, nil }

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

func init() {
	driver.Register[brightness.Driver](factory{})
}
