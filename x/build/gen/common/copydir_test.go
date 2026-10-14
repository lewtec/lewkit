package common

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyDir_PreservesTree(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "a.txt"), []byte("hello"), 0o644))

	dst := filepath.Join(t.TempDir(), "out")
	require.NoError(t, CopyDir(src, dst))
	got, err := os.ReadFile(filepath.Join(dst, "nested", "a.txt"))
	require.NoError(t, err)
	if string(got) != "hello" {
		require.FailNowf(t, "", "got %q", got)
	}
}

func TestReplaceDir_MovesOntoExisting(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.MkdirAll(dst, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0o644))

	require.NoError(t, ReplaceDir(src, dst))
	_, err := os.Stat(filepath.Join(dst, "old.txt"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	got, err := os.ReadFile(filepath.Join(dst, "new.txt"))
	require.NoError(t, err)
	if string(got) != "new" {
		require.FailNowf(t, "", "got %q", got)
	}
}

func TestCopyFile_WritesMode(t *testing.T) {
	src := filepath.Join(t.TempDir(), "built")
	require.NoError(t, os.WriteFile(src, []byte("exe"), 0o644))
	dst := filepath.Join(t.TempDir(), "dist", "app.exe")
	require.NoError(t, CopyFile(src, dst, 0o755))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	if string(got) != "exe" {
		require.FailNowf(t, "", "got %q", got)
	}
	info, err := os.Stat(dst)
	require.NoError(t, err)
	if info.Mode().Perm() != 0o755 {
		require.FailNowf(t, "", "mode=%v", info.Mode().Perm())
	}
}
