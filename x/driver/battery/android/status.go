package android

import "github.com/lewtec/lewkit/x/driver/battery"

// levels are BatteryManager status codes.
type levels struct {
	unknown     int
	charging    int
	discharging int
	notCharging int
	full        int
}

// percent maps EXTRA_LEVEL and EXTRA_SCALE onto 0..100.
func percent(level, scale int) (int, error) {
	if level < 0 || scale <= 0 {
		return 0, battery.ErrUnknownLevel
	}
	if level > scale {
		level = scale
	}
	return (level*100 + scale/2) / scale, nil
}

func statusFrom(present bool, code int, known levels) (battery.Status, error) {
	if !present {
		return battery.Unknown, battery.ErrNoBattery
	}
	switch code {
	case known.charging:
		return battery.Charging, nil
	case known.discharging:
		return battery.Discharging, nil
	case known.notCharging:
		return battery.Status("Not charging"), nil
	case known.full:
		return battery.Full, nil
	default:
		return battery.Unknown, nil
	}
}
