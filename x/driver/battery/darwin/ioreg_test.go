package darwin

import (
	"runtime"
	"strconv"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

type gauge struct {
	now, full, charging, charged, external, installed string
}

func TestParseIOReg(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		status   battery.Status
		level    int
		err      error
		levelErr error
	}{
		{
			name:   "discharging",
			in:     batteryText(gauge{now: "65", full: "100", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status: battery.Discharging,
			level:  65,
		},
		{
			name:   "charging",
			in:     batteryText(gauge{now: "40", full: "100", charging: "Yes", charged: "No", external: "Yes", installed: "Yes"}),
			status: battery.Charging,
			level:  40,
		},
		{
			name:   "full",
			in:     batteryText(gauge{now: "100", full: "100", charging: "No", charged: "Yes", external: "Yes", installed: "Yes"}),
			status: battery.Full,
			level:  100,
		},
		{
			name:   "not charging",
			in:     batteryText(gauge{now: "80", full: "100", charging: "No", charged: "No", external: "Yes", installed: "Yes"}),
			status: battery.Status("Not charging"),
			level:  80,
		},
		{
			name:   "ratio",
			in:     batteryText(gauge{now: "1", full: "3", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status: battery.Discharging,
			level:  33,
		},
		{
			name:   "clamp",
			in:     batteryText(gauge{now: "140", full: "100", charging: "Yes", charged: "No", external: "Yes", installed: "Yes"}),
			status: battery.Charging,
			level:  100,
		},
		{
			name:   "percent without max",
			in:     "\"CurrentCapacity\" = 66\n\"ExternalConnected\" = No\n",
			status: battery.Discharging,
			level:  66,
		},
		{
			name:     "raw without max",
			in:       "\"CurrentCapacity\" = 2500\n\"IsCharging\" = Yes\n",
			status:   battery.Charging,
			levelErr: battery.ErrUnknownLevel,
		},
		{
			name:     "zero max",
			in:       batteryText(gauge{now: "10", full: "0", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status:   battery.Discharging,
			levelErr: battery.ErrUnknownLevel,
		},
		{
			name:     "negative",
			in:       batteryText(gauge{now: "-1", full: "100", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status:   battery.Discharging,
			levelErr: battery.ErrUnknownLevel,
		},
		{
			name:     "bad capacity",
			in:       batteryText(gauge{now: "full", full: "100", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status:   battery.Discharging,
			levelErr: strconv.ErrSyntax,
		},
		{
			name:     "missing capacity",
			in:       "\"IsCharging\" = Yes\n\"BatteryInstalled\" = Yes\n",
			status:   battery.Charging,
			levelErr: battery.ErrUnknownLevel,
		},
		{
			name:   "unknown power source",
			in:     "\"CurrentCapacity\" = 50\n\"MaxCapacity\" = 100\n\"BatteryInstalled\" = Yes\n",
			status: battery.Unknown,
			level:  50,
		},
		{
			name: "nested blob does not override",
			in: "\"BatteryData\" = {\"CurrentCapacity\"=1,\"MaxCapacity\"=2}\n" +
				batteryText(gauge{now: "65", full: "100", charging: "No", charged: "No", external: "No", installed: "Yes"}),
			status: battery.Discharging,
			level:  65,
		},
		{name: "empty", in: "", err: battery.ErrNoBattery},
		{name: "removed", in: "\"BatteryInstalled\" = No\n\"CurrentCapacity\" = 10\n", err: battery.ErrNoBattery},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIOReg(tt.in)
			require.ErrorIs(t, err, tt.err)
			if tt.err != nil {
				return
			}
			require.Equal(t, tt.status, got.status)
			require.ErrorIs(t, got.levelErr, tt.levelErr)
			if tt.levelErr == nil {
				require.Equal(t, tt.level, got.level)
			}
		})
	}
}

func TestNotDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func batteryText(g gauge) string {
	return "" +
		"\"CurrentCapacity\" = " + g.now + "\n" +
		"\"MaxCapacity\" = " + g.full + "\n" +
		"\"IsCharging\" = " + g.charging + "\n" +
		"\"FullyCharged\" = " + g.charged + "\n" +
		"\"ExternalConnected\" = " + g.external + "\n" +
		"\"BatteryInstalled\" = " + g.installed + "\n"
}
