// Package windows reads battery status and charge level from GetSystemPowerStatus.
// No system battery is ErrNoBattery. A percent of 255 is ErrUnknownLevel.
package windows

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
)

type factory struct{}

func (factory) ID() string   { return "battery_windows" }
func (factory) Name() string { return "GetSystemPowerStatus" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	_, _, err := sample()
	if errors.Is(err, battery.ErrUnknownLevel) {
		return nil
	}
	return err
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) BatteryStatus(ctx context.Context) (battery.Status, error) {
	if err := ctx.Err(); err != nil {
		return battery.Unknown, err
	}
	status, _, err := sample()
	if errors.Is(err, battery.ErrUnknownLevel) {
		return status, nil
	}
	return status, err
}

func (backend) BatteryLevel(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	_, level, err := sample()
	return level, err
}

func sample() (battery.Status, int, error) {
	ac, flag, percent, err := readPower()
	if err != nil {
		return battery.Unknown, 0, err
	}
	return interpret(ac, flag, percent)
}

func init() {
	driver.Register[battery.Driver](factory{})
}
