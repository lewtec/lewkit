package linux

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	if parseStatus("Charging\n") != battery.Charging {
		require.FailNow(t, fmt.Sprint("charging"))
	}
	if parseStatus("Discharging") != battery.Discharging {
		require.FailNow(t, fmt.Sprint("discharging"))
	}
	if parseStatus("Full") != battery.Full {
		require.FailNow(t, fmt.Sprint("full"))
	}
	if parseStatus("Not charging\n") != battery.Status("Not charging") {
		require.FailNow(t, fmt.Sprint("other"))
	}
}
