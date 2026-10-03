package windows

import "github.com/lewtec/lewkit/x/driver/battery"

const (
	batteryCharging = 8
	batteryNone     = 128
	percentUnknown  = 255
	acOnline        = 1
	acOffline       = 0
)

func interpret(ac, flag, percent byte) (battery.Status, int, error) {
	if flag&batteryNone != 0 {
		return battery.Unknown, 0, battery.ErrNoBattery
	}
	var status battery.Status
	switch {
	case flag&batteryCharging != 0:
		status = battery.Charging
	case percent == 100 && ac == acOnline:
		status = battery.Full
	case ac == acOffline:
		status = battery.Discharging
	default:
		status = battery.Unknown
	}
	if percent == percentUnknown || percent > 100 {
		return status, 0, battery.ErrUnknownLevel
	}
	return status, int(percent), nil
}
