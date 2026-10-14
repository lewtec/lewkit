package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStampPackagingVersion_FillsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	vi, name, code := StampPackagingVersion(t.Context(), dir, "", 0)
	if vi.Version == "" {
		require.FailNow(t, fmt.Sprint("empty version info"))
	}
	if name == "" {
		require.FailNow(t, fmt.Sprint("empty name"))
	}
	if code <= 0 {
		require.FailNowf(t, "", "code=%d", code)
	}
}

func TestStampPackagingVersion_KeepsExplicit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, name, code := StampPackagingVersion(t.Context(), dir, "9.8.7", 42)
	if name != "9.8.7" {
		require.FailNowf(t, "", "name=%q", name)
	}
	if code != 42 {
		require.FailNowf(t, "", "code=%d", code)
	}
}
