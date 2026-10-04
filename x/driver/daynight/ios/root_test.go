package ios

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
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

func TestCurrentAndWatch(t *testing.T) {
	watchEvery = 20 * time.Millisecond
	t.Cleanup(func() { watchEvery = 400 * time.Millisecond })
	dir := t.TempDir()
	base := iosbox.WithDir(t.Context(), dir)
	file := filepath.Join(dir, "daynight.txt")
	if err := os.WriteFile(file, []byte("light\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mode, err := backend{}.Current(base)
	if err != nil || mode != daynight.Light {
		t.Fatalf("%s %v", mode, err)
	}
	ctx, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()
	changes, err := backend{}.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := <-changes; got != daynight.Light {
		t.Fatalf("first %s", got)
	}
	if err := os.WriteFile(file, []byte("dark\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-changes:
		if got != daynight.Dark {
			t.Fatalf("change %s", got)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
