package path

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadMapFS(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.txt":     {Data: []byte("hi")},
		"dir/b.txt": {Data: []byte("b")},
	}
	p := New("a.txt")
	b, err := p.ReadFile(fsys)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)

	st, err := p.Stat(fsys)
	require.NoError(t, err)
	assert.False(t, st.IsDir())

	ents, err := New("dir").ReadDir(fsys)
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "b.txt", ents[0].Name())

	assert.Equal(t, []string{"dir/b.txt"}, names(test.Collect(t, New("dir").Glob(fsys, "*.txt"))))

	walked := names(test.Collect(t, New(".").Walk(fsys)))
	assert.Contains(t, walked, "a.txt")
	assert.Contains(t, walked, "dir/b.txt")

	assert.Equal(t, []Path{New("dir/b.txt")}, test.Collect(t, New("dir").IterDir(fsys)))
}

func TestPredicates(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.txt": {Data: []byte("hi")},
		"dir":   {Mode: fs.ModeDir},
	}
	ok, err := New("a.txt").Exists(fsys)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = New("nope").Exists(fsys)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = New("dir").IsDir(fsys)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = New("a.txt").IsDir(fsys)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = New("nope").IsDir(fsys)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = New("a.txt").IsFile(fsys)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = New("dir").IsFile(fsys)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{}
	p := New("a.txt")
	err := p.WriteFile(fsys, []byte("x"), 0o644)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReadOnly)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "a.txt", pe.Path)

	_, err = p.Create(fsys)
	assert.ErrorIs(t, err, ErrReadOnly)
	assert.ErrorIs(t, p.Mkdir(fsys, 0o755), ErrReadOnly)
	assert.ErrorIs(t, p.Remove(fsys), ErrReadOnly)
	assert.ErrorIs(t, p.Rename(fsys, New("b")), ErrReadOnly)
	assert.ErrorIs(t, p.Symlink(fsys, New("b")), ErrReadOnly)
	assert.ErrorIs(t, p.Hardlink(fsys, New("b")), ErrReadOnly)
	assert.ErrorIs(t, p.Chmod(fsys, 0o600), ErrReadOnly)
	_, err = p.OpenRoot(fsys)
	assert.ErrorIs(t, err, ErrReadOnly)
}

func TestOpenRoot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := dir + "/xpath-file"
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	_, err := Open("")
	assert.ErrorIs(t, err, ErrEmptyPath)

	_, err = Open(file)
	assert.ErrorIs(t, err, ErrNotDir)

	_, err = Open(dir + "/missing")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	root, err := Open(dir)
	require.NoError(t, err)
	test.CloseOnCleanup(t, root)
	assert.Equal(t, dir, root.Name())
}

func TestRootIO(t *testing.T) {
	t.Parallel()
	root, err := Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, root)

	p := New("a.txt")
	require.NoError(t, p.WriteFile(root, []byte("hi"), 0o644))
	b, err := p.ReadFile(root)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)

	f, err := New("c.txt").Create(root)
	require.NoError(t, err)
	w, ok := f.(io.Writer)
	require.True(t, ok)
	_, err = w.Write([]byte("c"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	b, err = New("c.txt").ReadFile(root)
	require.NoError(t, err)
	assert.Equal(t, []byte("c"), b)

	require.NoError(t, New("sub").MkdirAll(root, 0o755))
	require.NoError(t, New("sub", "d.txt").WriteFile(root, nil, 0o644))
	okDir, err := New("sub").IsDir(root)
	require.NoError(t, err)
	assert.True(t, okDir)

	require.NoError(t, New("a.txt").Rename(root, New("b.txt")))
	okEx, err := New("a.txt").Exists(root)
	require.NoError(t, err)
	assert.False(t, okEx)
	okEx, err = New("b.txt").Exists(root)
	require.NoError(t, err)
	assert.True(t, okEx)

	link := New("link")
	require.NoError(t, link.Symlink(root, New("b.txt")))
	okLink, err := link.IsSymlink(root)
	require.NoError(t, err)
	assert.True(t, okLink)
	target, err := link.ReadLink(root)
	require.NoError(t, err)
	assert.Equal(t, "b.txt", target.String())

	hard := New("hard")
	require.NoError(t, hard.Hardlink(root, New("b.txt")))
	b, err = hard.ReadFile(root)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)

	require.NoError(t, New("b.txt").Chmod(root, 0o600))
	st, err := New("b.txt").Stat(root)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o600), st.Mode().Perm())

	nested, err := New("sub").OpenRoot(root)
	require.NoError(t, err)
	test.CloseOnCleanup(t, nested)
	b, err = New("d.txt").ReadFile(nested)
	require.NoError(t, err)
	assert.Empty(t, b)

	subFS, err := fs.Sub(root, "sub")
	require.NoError(t, err)
	ents, err := New(".").ReadDir(subFS)
	require.NoError(t, err)
	names := make([]string, len(ents))
	for i, e := range ents {
		names[i] = e.Name()
	}
	assert.True(t, slices.Contains(names, "d.txt"))

	require.NoError(t, New("c.txt").Remove(root))
	okEx, err = New("c.txt").Exists(root)
	require.NoError(t, err)
	assert.False(t, okEx)

	require.NoError(t, New("sub").RemoveAll(root))
	okEx, err = New("sub").Exists(root)
	require.NoError(t, err)
	assert.False(t, okEx)
}

func TestPathOpen(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a.txt": {Data: []byte("hi")}}
	f, err := New("a.txt").Open(fsys)
	require.NoError(t, err)
	test.CloseOnCleanup(t, f)
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
}
