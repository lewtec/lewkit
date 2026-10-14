package windows

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

func TestInterpret(t *testing.T) {
	status, level, err := interpret(acOnline, batteryCharging, 40)
	if err != nil || status != battery.Charging || level != 40 {
		require.FailNowf(t, "", "charging: %v %d %v", status, level, err)
	}
	status, level, err = interpret(acOffline, 0, 42)
	if err != nil || status != battery.Discharging || level != 42 {
		require.FailNowf(t, "", "discharging: %v %d %v", status, level, err)
	}
	status, level, err = interpret(acOnline, 0, 100)
	if err != nil || status != battery.Full || level != 100 {
		require.FailNowf(t, "", "full: %v %d %v", status, level, err)
	}
	_, _, err = interpret(acOnline, batteryNone, percentUnknown)
	require.ErrorIs(t, err, battery.ErrNoBattery)
	status, _, err = interpret(acOffline, 0, percentUnknown)
	if !errors.Is(err, battery.ErrUnknownLevel) || status != battery.Discharging {
		require.FailNowf(t, "", "unknown level: %v %v", status, err)
	}
}
