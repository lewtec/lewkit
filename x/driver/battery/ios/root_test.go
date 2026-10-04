package ios

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

func TestNotIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatal(err)
	}
}

func TestReadsHostFile(t *testing.T) {
	dir := t.TempDir()
	ctx := iosbox.WithDir(t.Context(), dir)
	body := []byte(`{"status":"Discharging","level":-1}`)
	if err := os.WriteFile(filepath.Join(dir, "battery.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := backend{}.BatteryStatus(ctx)
	if err != nil || status != battery.Discharging {
		t.Fatalf("%s %v", status, err)
	}
	_, err = backend{}.BatteryLevel(ctx)
	if !errors.Is(err, battery.ErrUnknownLevel) {
		t.Fatal(err)
	}
}
