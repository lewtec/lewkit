// Package battery reads the system battery status.
//
//	status, err := battery.BatteryStatus(ctx)
//	if errors.Is(err, battery.ErrNoBattery) {
//
// Import [github.com/lewtec/lewkit/x/driver/battery/linux]. The backend
// reads the first /sys/class/power_supply/BAT* status file.
package battery

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
)

// ErrNoBattery means sysfs has no BAT* supply.
var ErrNoBattery = errors.New("no battery found")

// Status is the kernel power_supply status string.
type Status string

const (
	Charging    Status = "Charging"
	Discharging Status = "Discharging"
	Full        Status = "Full"
	Unknown     Status = "Unknown"
)

// Driver reads battery status.
type Driver interface {
	BatteryStatus(ctx context.Context) (Status, error)
}

// BatteryStatus returns the active battery status.
func BatteryStatus(ctx context.Context) (Status, error) {
	return driver.WithResult(ctx, func(d Driver) (Status, error) {
		return d.BatteryStatus(ctx)
	})
}
