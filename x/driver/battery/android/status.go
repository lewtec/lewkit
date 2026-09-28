package android

import (
	"strings"

	"github.com/lewtec/lewkit/x/driver/battery"
)

func statusFromHost(text string) (battery.Status, error) {
	if strings.TrimSpace(text) == "" {
		return battery.Unknown, battery.ErrNoBattery
	}
	return parseStatus(text), nil
}

func parseStatus(text string) battery.Status {
	switch strings.TrimSpace(text) {
	case "Charging":
		return battery.Charging
	case "Discharging":
		return battery.Discharging
	case "Full":
		return battery.Full
	default:
		return battery.Status(strings.TrimSpace(text))
	}
}
