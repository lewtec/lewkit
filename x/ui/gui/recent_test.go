package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecentRemember(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	first := t.TempDir()
	second := t.TempDir()
	require.NoError(t, Remember(first))
	require.NoError(t, Remember(second))
	require.NoError(t, Remember(first))
	got, err := Recent()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, absPath(t, first), got[0].Path)
	assert.Equal(t, absPath(t, second), got[1].Path)
}

func TestRecentMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Recent()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRecentSkipsMissingDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	keep := t.TempDir()
	require.NoError(t, Remember(keep))
	file := filepath.Join(home, "lewkit", "recent-dirs")
	require.NoError(t, os.WriteFile(file, []byte("/no/such/lewkit/dir\n"+absPath(t, keep)+"\n"), 0o600))
	got, err := Recent()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, absPath(t, keep), got[0].Path)
}

func absPath(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	require.NoError(t, err)
	return abs
}
