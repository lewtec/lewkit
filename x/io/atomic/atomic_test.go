package atomic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitReplacesDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "tool")
	require.NoError(t, os.MkdirAll(filepath.Join(destination, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destination, "bin", "old"), []byte("old"), 0o644))

	operation := NewOperation(destination, true)
	require.NoError(t, os.MkdirAll(filepath.Join(operation.StagingPath(), "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(operation.StagingPath(), "bin", "new"), []byte("new"), 0o644))
	require.NoError(t, operation.Commit())

	body, err := os.ReadFile(filepath.Join(destination, "bin", "new"))
	require.NoError(t, err)
	require.Equal(t, "new", string(body))
	_, err = os.Stat(filepath.Join(destination, "bin", "old"))
	require.ErrorIs(t, err, os.ErrNotExist)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Equal(t, []string{"tool"}, names(entries))
}

func TestCommitCreatesMissingDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "tool")
	operation := NewOperation(destination, true)
	require.NoError(t, os.MkdirAll(operation.StagingPath(), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(operation.StagingPath(), "marker"), []byte("ok"), 0o644))
	require.NoError(t, operation.Commit())
	body, err := os.ReadFile(filepath.Join(destination, "marker"))
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

func TestWriteStringReplacesFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "note")
	require.NoError(t, WriteString(path, "one"))
	require.NoError(t, WriteString(path, "two"))
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "two", string(body))
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Equal(t, []string{"note"}, names(entries))
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
}
