package windows

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
)

func TestInterpret(t *testing.T) {
	status, level, err := interpret(acOnline, batteryCharging, 40)
	if err != nil || status != battery.Charging || level != 40 {
		t.Fatalf("charging: %v %d %v", status, level, err)
	}
	status, level, err = interpret(acOffline, 0, 42)
	if err != nil || status != battery.Discharging || level != 42 {
		t.Fatalf("discharging: %v %d %v", status, level, err)
	}
	status, level, err = interpret(acOnline, 0, 100)
	if err != nil || status != battery.Full || level != 100 {
		t.Fatalf("full: %v %d %v", status, level, err)
	}
	_, _, err = interpret(acOnline, batteryNone, percentUnknown)
	if !errors.Is(err, battery.ErrNoBattery) {
		t.Fatalf("no battery: %v", err)
	}
	status, _, err = interpret(acOffline, 0, percentUnknown)
	if !errors.Is(err, battery.ErrUnknownLevel) || status != battery.Discharging {
		t.Fatalf("unknown level: %v %v", status, err)
	}
}
