//go:build windows

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatOutputAcceptsExeSibling(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.exe"), []byte("x"), 0o644))
	info, err := StatOutput(filepath.Join(dir, "hello"))
	require.NoError(t, err)
	require.Equal(t, "hello.exe", info.Name())

	_, err = StatOutput(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
