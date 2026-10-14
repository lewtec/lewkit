package ios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestNotIOS(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, "ios")
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
