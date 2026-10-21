package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
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

func TestStampPackagingVersion_UsesCommitCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		full := append([]string{
			"-c", "user.email=test@example.com",
			"-c", "user.name=test",
			"-c", "commit.gpgsign=false",
		}, args...)
		cmd := execdriver.MustCommand(t.Context(), "git", full...)
		cmd.Dir = dir
		require.NoError(t, execdriver.Run(t.Context(), cmd))
	}
	git("init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	git("add", "a.txt")
	git("commit", "-m", "one")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644))
	git("add", "b.txt")
	git("commit", "-m", "two")
	git("tag", "v1.2.3")
	_, name, code := StampPackagingVersion(t.Context(), dir, "", 0)
	require.Equal(t, "1.2.3", name)
	require.Equal(t, 2, code)
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
