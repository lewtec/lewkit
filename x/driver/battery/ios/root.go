// Package ios reads battery status from the packaged iOS host.
// The host writes battery.json. Level is 0..100. A negative level is unknown.
package ios

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

type factory struct{}

func (factory) ID() string   { return "battery_ios" }
func (factory) Name() string { return "iOS battery" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if !iosbox.Available() {
		return fmt.Errorf("%w: no ios host", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) BatteryStatus(ctx context.Context) (battery.Status, error) {
	if err := ctx.Err(); err != nil {
		return battery.Unknown, err
	}
	got, err := iosbox.ReadBattery(ctx)
	if err != nil {
		return battery.Unknown, err
	}
	return statusOf(got.Status), nil
}

func (backend) BatteryLevel(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	got, err := iosbox.ReadBattery(ctx)
	if err != nil {
		return 0, err
	}
	if got.Level < 0 || got.Level > 100 {
		return 0, battery.ErrUnknownLevel
	}
	return got.Level, nil
}

func statusOf(text string) battery.Status {
	switch battery.Status(text) {
	case battery.Charging, battery.Discharging, battery.Full, battery.Unknown:
		return battery.Status(text)
	default:
		return battery.Unknown
	}
}

func init() { driver.Register[battery.Driver](factory{}) }

var (
	_ driver.DriverFactory[battery.Driver] = factory{}
	_ driver.Weighter                      = factory{}
)
