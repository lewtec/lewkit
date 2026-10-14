package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveWorkDir_CreatesMissingPath(t *testing.T) {
	want := filepath.Join(t.TempDir(), "nested", "work")
	got, cleanup, err := ResolveWorkDir(want, "eletrocromo-test-*")
	require.NoError(t, err)
	if cleanup {
		require.FailNow(t, fmt.Sprint("explicit work dir should not be marked cleanup"))
	}
	abs, err := filepath.Abs(want)
	require.NoError(t, err)
	if got != abs {
		require.FailNowf(t, "", "got %q want %q", got, abs)
	}
	st, err := os.Stat(got)
	require.NoError(t, err)
	if !st.IsDir() {
		require.FailNowf(t, "", "%s is not a directory", got)
	}
}

func TestResolveWorkDir_EmptyMakesTemp(t *testing.T) {
	dir, cleanup, err := ResolveWorkDir("", "eletrocromo-test-*")
	require.NoError(t, err)
	if !cleanup {
		require.FailNow(t, fmt.Sprint("temp work dir should be marked cleanup"))
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("cleanup temp work dir: %v", err)
		}
	})
	if !strings.Contains(filepath.Base(dir), "eletrocromo-test-") {
		require.FailNowf(t, "", "temp name %q missing prefix", dir)
	}
	st, err := os.Stat(dir)
	require.NoError(t, err)
	if !st.IsDir() {
		require.FailNowf(t, "", "%s is not a directory", dir)
	}
}
