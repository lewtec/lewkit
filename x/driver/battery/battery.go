// Package battery reads the system battery status and charge level.
//
//	status, err := battery.BatteryStatus(ctx)
//	level, err := battery.BatteryLevel(ctx)
//	if errors.Is(err, battery.ErrNoBattery) {
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one backend.
// Linux reads the first /sys/class/power_supply/BAT* supply.
// Darwin reads AppleSmartBattery through ioreg. Level is 0..100.
package battery

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
)

// ErrNoBattery means no battery supply is present.
var ErrNoBattery = errors.New("no battery found")

// ErrUnknownLevel means the supply has no charge reading from 0 to 100.
var ErrUnknownLevel = errors.New("battery level unknown")

// Status is the kernel power_supply status string.
type Status string

const (
	Charging    Status = "Charging"
	Discharging Status = "Discharging"
	Full        Status = "Full"
	Unknown     Status = "Unknown"
)

// Driver reads battery status and charge level.
type Driver interface {
	BatteryStatus(ctx context.Context) (Status, error)
	BatteryLevel(ctx context.Context) (int, error)
}

// BatteryStatus returns the active battery status.
func BatteryStatus(ctx context.Context) (Status, error) {
	return driver.WithResult(ctx, func(d Driver) (Status, error) {
		return d.BatteryStatus(ctx)
	})
}

// BatteryLevel returns the charge percentage from 0 to 100.
func BatteryLevel(ctx context.Context) (int, error) {
	return driver.WithResult(ctx, func(d Driver) (int, error) {
		return d.BatteryLevel(ctx)
	})
}
