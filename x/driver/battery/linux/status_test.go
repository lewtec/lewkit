package linux

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
)

func TestParseStatus(t *testing.T) {
	if parseStatus("Charging\n") != battery.Charging {
		t.Fatal("charging")
	}
	if parseStatus("Discharging") != battery.Discharging {
		t.Fatal("discharging")
	}
	if parseStatus("Full") != battery.Full {
		t.Fatal("full")
	}
	if parseStatus("Not charging\n") != battery.Status("Not charging") {
		t.Fatal("other")
	}
}
