package path

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripTopLevelDirectory(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	nested := filepath.Join(directory, "demo-1.0.0", "bin")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "demo"), []byte("payload"), 0o755))

	root, err := Open(directory)
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })
	require.NoError(t, StripTopLevelDirectory(root))

	body, err := os.ReadFile(filepath.Join(directory, "bin", "demo"))
	require.NoError(t, err)
	require.Equal(t, "payload", string(body))
	_, err = os.Stat(filepath.Join(directory, "demo-1.0.0"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestStripTopLevelDirectoryLeavesMixedRoot(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "keep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "readme"), []byte("hi"), 0o644))

	root, err := Open(directory)
	require.NoError(t, err)
	t.Cleanup(func() { root.Close() })
	require.NoError(t, StripTopLevelDirectory(root))

	_, err = os.Stat(filepath.Join(directory, "keep"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(directory, "readme"))
	require.NoError(t, err)
}
