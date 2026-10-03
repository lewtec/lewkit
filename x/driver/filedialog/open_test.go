package filedialog

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestOpenDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "bb")

	fsys, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a.txt", "sub", "sub/b.txt"))
	body, err := fs.ReadFile(fsys, "sub/b.txt")
	require.NoError(t, err)
	require.Equal(t, "bb", string(body))
	_, err = fsys.Open("..")
	require.ErrorIs(t, err, fs.ErrInvalid)
}

func TestOpenFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	writeFile(t, path, "hi")

	fsys, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "notes.txt"))
	body, err := fs.ReadFile(fsys, "notes.txt")
	require.NoError(t, err)
	require.Equal(t, "hi", string(body))
}

func TestOpenSeveral(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeFile(t, filepath.Join(docs, "a.txt"), "a")
	note := filepath.Join(root, "note.txt")
	writeFile(t, note, "one")
	again := filepath.Join(root, "other", "note.txt")
	writeFile(t, again, "two")
	taken := filepath.Join(root, "taken", "note (2).txt")
	writeFile(t, taken, "three")

	fsys, err := Open(docs, note, taken, again)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "docs/a.txt", "note.txt", "note (2).txt", "note (3).txt"))
	body, err := fs.ReadFile(fsys, "note (3).txt")
	require.NoError(t, err)
	require.Equal(t, "two", string(body))
}

func TestOpenRejects(t *testing.T) {
	t.Parallel()
	_, err := Open()
	require.ErrorIs(t, err, ErrRequest)
	_, err = Open("content://a", filepath.Join(t.TempDir(), "local"))
	require.ErrorIs(t, err, ErrRequest)
	_, err = Open(filepath.Join(t.TempDir(), "missing"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestOpenContent(t *testing.T) {
	_, err := Open("content://tree")
	require.ErrorIs(t, err, driver.ErrUnavailable)
	require.ErrorIs(t, err, errContent)

	RegisterContent(func(names []string) (fs.FS, error) {
		require.Equal(t, []string{"content://tree"}, names)
		return fstest.MapFS{"a.txt": &fstest.MapFile{Data: []byte("a")}}, nil
	})
	t.Cleanup(func() { RegisterContent(nil) })

	fsys, err := Open("content://tree")
	require.NoError(t, err)
	body, err := fs.ReadFile(fsys, "a.txt")
	require.NoError(t, err)
	require.Equal(t, "a", string(body))
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
