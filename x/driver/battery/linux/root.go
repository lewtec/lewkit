// Package linux reads battery status from sysfs.
package linux

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
)

type factory struct{}

func (factory) ID() string   { return "battery_linux" }
func (factory) Name() string { return "Linux sysfs" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*/status")
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("%w: /sys/class/power_supply/BAT*/status", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[battery.Driver](factory{})
}
