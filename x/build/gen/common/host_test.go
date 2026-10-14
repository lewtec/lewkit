package common

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkingDir(t *testing.T) {
	got, err := WorkingDir("  /tmp  ")
	require.NoError(t, err)
	require.Equal(t, "/tmp", got)
	got, err = WorkingDir(" ")
	require.NoError(t, err)
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, wd, got)
}

func TestWriters(t *testing.T) {
	stdout, stderr := Writers(t.Context(), nil, nil)
	require.NotNil(t, stdout)
	require.Equal(t, stdout, stderr)
	custom := &bytes.Buffer{}
	stdout, stderr = Writers(t.Context(), custom, nil)
	require.Equal(t, custom, stdout)
	require.Equal(t, custom, stderr)
}

func TestCleanupWorkDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "work")
	require.NoError(t, os.Mkdir(dir, 0o755))
	CleanupWorkDir(true, errors.New("failed"), false, false, dir, io.Discard, "test")
	_, err := os.Stat(dir)
	require.NoError(t, err)
	CleanupWorkDir(true, nil, true, false, dir, io.Discard, "test")
	_, err = os.Stat(dir)
	require.NoError(t, err)
	CleanupWorkDir(true, nil, false, false, dir, io.Discard, "test")
	_, err = os.Stat(dir)
	require.Error(t, err)
}

func TestIconRootKeepsExplicitDir(t *testing.T) {
	got, err := IconRoot("/base", "", "  given  ", "/work", "icons")
	require.NoError(t, err)
	require.Equal(t, "given", got)
}

func TestIconRootGenerates(t *testing.T) {
	work := t.TempDir()
	got, err := IconRoot(t.TempDir(), "", " ", work, "icons")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(work, "icons"), got)
	info, err := os.Stat(got)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}
