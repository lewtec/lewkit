//go:build darwin

package darwin

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestLiveAppleSmartBattery(t *testing.T) {
	test.DiscardSlog(t)
	ctx := t.Context()
	if _, err := read(ctx); errors.Is(err, battery.ErrNoBattery) {
		t.Skip(err.Error())
	}
	err := factory{}.CheckCompatibility(ctx)
	require.NoError(t, err)

	status, err := battery.BatteryStatus(ctx)
	require.NoError(t, err)
	level, err := battery.BatteryLevel(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, level, 0)
	require.LessOrEqual(t, level, 100)

	out, err := execdriver.Output(ctx, execdriver.MustCommand(ctx, "/usr/bin/pmset", "-g", "batt"))
	require.NoError(t, err)
	text := string(out)
	wantLevel, err := pmsetPercent(text)
	require.NoError(t, err)
	require.InDelta(t, float64(wantLevel), float64(level), 1)
	require.Equal(t, pmsetStatus(text), status)
	t.Logf("status=%s level=%d%%", status, level)
}

func pmsetPercent(text string) (int, error) {
	i := strings.Index(text, "%")
	if i < 0 {
		return 0, strconv.ErrSyntax
	}
	start := i
	for start > 0 && text[start-1] >= '0' && text[start-1] <= '9' {
		start--
	}
	if start == i {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(text[start:i])
}

func pmsetStatus(text string) battery.Status {
	line := strings.ToLower(text)
	switch {
	case strings.Contains(line, "discharging"):
		return battery.Discharging
	case strings.Contains(line, "not charging"):
		return battery.Status("Not charging")
	case strings.Contains(line, "finishing charge"), strings.Contains(line, "charging"):
		return battery.Charging
	case strings.Contains(line, "charged"):
		return battery.Full
	default:
		return battery.Unknown
	}
}
