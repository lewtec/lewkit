// Package linux reads battery status and charge level from sysfs.
// Level is capacity, then energy_now/energy_full, then charge_now/charge_full.
package linux

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
)

type factory struct{}

func (factory) ID() string   { return "battery_linux" }
func (factory) Name() string { return "Linux sysfs" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	_, err := batteryDir()
	if err == nil {
		return nil
	}
	if errors.Is(err, battery.ErrNoBattery) {
		return fmt.Errorf("%w: /sys/class/power_supply/BAT*/status", driver.ErrIncompatible)
	}
	return err
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[battery.Driver](factory{})
}
