package pulse

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/require"
)

func TestParseVolume(t *testing.T) {
	got, err := parseVolume("Volume: front-left: 26214 /  40% / -23.88 dB")
	if err != nil || got != 0.4 {
		require.FailNowf(t, "", "got %v %v", got, err)
	}
	got, err = parseVolume("no percent here")
	if err != nil || got != 0 {
		require.FailNowf(t, "", "missing percent: %v %v", got, err)
	}
	if _, err := parseVolume("xx%"); err == nil {
		require.FailNow(t, fmt.Sprint("expected atoi error"))
	}
}

func TestMissingBinary(t *testing.T) {
	err := execdriver.RequireBinary(t.Context(), "lewkit-missing-pactl")
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
