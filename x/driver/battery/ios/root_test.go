package ios

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestNotIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		test.Skip(t)
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestReadsHostFile(t *testing.T) {
	dir := t.TempDir()
	ctx := iosbox.WithDir(t.Context(), dir)
	body := []byte(`{"status":"Discharging","level":-1}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "battery.json"), body, 0o600))
	status, err := backend{}.BatteryStatus(ctx)
	if err != nil || status != battery.Discharging {
		require.FailNowf(t, "", "%s %v", status, err)
	}
	_, err = backend{}.BatteryLevel(ctx)
	require.ErrorIs(t, err, battery.ErrUnknownLevel)
}
