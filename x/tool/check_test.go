package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaryCandidates(t *testing.T) {
	base := filepath.Join("tools", "pkg", "1.0.0")
	got := BinaryCandidates(base, "gh")
	want := []string{
		filepath.Join(base, "bin", "gh"),
		filepath.Join(base, "bin", "gh.exe"),
		filepath.Join(base, "bin", "gh.cmd"),
		filepath.Join(base, "bin", "gh.bat"),
		filepath.Join(base, "gh"),
		filepath.Join(base, "gh.exe"),
		filepath.Join(base, "gh.cmd"),
		filepath.Join(base, "gh.bat"),
	}
	require.Equal(t, want, got)
}

func TestCheckRejectsParent(t *testing.T) {
	err := FileExists("../outside").Check(t.Context(), t.TempDir())
	require.ErrorIs(t, err, ErrPathEscapes)
}

func TestCheckEmptyRelativePath(t *testing.T) {
	err := FileExists(".").Check(t.Context(), t.TempDir())
	require.ErrorIs(t, err, ErrEmptyRelativePath)
}

func TestBinaryCheck(t *testing.T) {
	destination := t.TempDir()
	directory := filepath.Join(destination, "bin")
	require.NoError(t, os.MkdirAll(directory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "demo"), []byte("ok"), 0o755))
	require.NoError(t, Binary("demo").Check(t.Context(), destination))
	require.ErrorIs(t, Binary("missing").Check(t.Context(), destination), ErrBinaryNotFound)
}

func TestFindBinaryAcceptsSymlinkOutsideTree(t *testing.T) {
	destination := t.TempDir()
	targetDirectory := t.TempDir()
	target := filepath.Join(targetDirectory, "node")
	require.NoError(t, os.WriteFile(target, []byte("ok"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(destination, "bin"), 0o755))
	link := filepath.Join(destination, "bin", "node")
	require.NoError(t, os.Symlink(target, link))
	require.Equal(t, link, FindBinary(destination, "node"))
	require.NoError(t, Binary("node").Check(t.Context(), destination))
}
