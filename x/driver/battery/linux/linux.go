package linux

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver/battery"
)

type backend struct{}

func (backend) BatteryStatus(context.Context) (battery.Status, error) {
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*/status")
	if err != nil {
		return battery.Unknown, err
	}
	if len(matches) == 0 {
		return battery.Unknown, battery.ErrNoBattery
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return battery.Unknown, err
	}
	return parseStatus(string(data)), nil
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
