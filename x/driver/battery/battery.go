// Package battery reads the system battery status and charge level.
//
//	status, err := battery.BatteryStatus(ctx)
//	level, err := battery.BatteryLevel(ctx)
//	if errors.Is(err, battery.ErrNoBattery) {
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one backend.
// Linux reads the first /sys/class/power_supply/BAT* supply.
// Darwin reads AppleSmartBattery through ioreg. iOS reads the host
// battery file. Windows reads GetSystemPowerStatus. Level is 0..100.
package battery

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/errs"
)

// ErrNoBattery means no battery supply is present.
var ErrNoBattery = errs.Wrap(errs.ErrNotFound, "no battery found")

// ErrUnknownLevel means the supply has no charge reading from 0 to 100.
var ErrUnknownLevel = errs.Wrap(errs.ErrUnknown, "battery level unknown")

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

// Percent maps a charge gauge to 0..100.
// now above full counts as full. A negative now or a non-positive full is [ErrUnknownLevel].
func Percent(now, full int64) (int, error) {
	if now < 0 || full <= 0 {
		return 0, ErrUnknownLevel
	}
	if now > full {
		now = full
	}
	return int((now*100 + full/2) / full), nil
}
