package ios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestNotIOS(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, "ios")
}

func TestCurrentAndWatch(t *testing.T) {
	watchEvery = 20 * time.Millisecond
	t.Cleanup(func() { watchEvery = 400 * time.Millisecond })
	dir := t.TempDir()
	base := iosbox.WithDir(t.Context(), dir)
	file := filepath.Join(dir, "daynight.txt")
	require.NoError(t, os.WriteFile(file, []byte("light\n"), 0o600))
	mode, err := backend{}.Current(base)
	if err != nil || mode != daynight.Light {
		require.FailNowf(t, "", "%s %v", mode, err)
	}
	ctx, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()
	changes, err := backend{}.Watch(ctx)
	require.NoError(t, err)
	if got := <-changes; got != daynight.Light {
		require.FailNowf(t, "", "first %s", got)
	}
	require.NoError(t, os.WriteFile(file, []byte("dark\n"), 0o600))
	select {
	case got := <-changes:
		if got != daynight.Dark {
			require.FailNowf(t, "", "change %s", got)
		}
	case <-ctx.Done():
		require.FailNow(t, fmt.Sprint(ctx.Err()))
	}
}
