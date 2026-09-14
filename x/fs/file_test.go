package fs

import (
	"io"
	iofs "io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func memFile(name string, data []byte) File {
	host := fstest.MapFS{name: {Data: data, Mode: 0o644}}
	return File{
		Name: path.New(name),
		Mode: 0o644,
		Size: int64(len(data)),
		Open: func() (iofs.File, error) {
			return host.Open(name)
		},
	}
}

func memDir(name string) File {
	return File{Name: path.New(name), Mode: iofs.ModeDir | 0o555}
}

func listing(files ...File) Files {
	return func(yield func(File, error) bool) {
		for _, f := range files {
			if !yield(f, nil) {
				return
			}
		}
	}
}

func TestNames(t *testing.T) {
	t.Parallel()
	got := test.Collect(t, Names(listing(memFile("a/b.txt", []byte("x")), memDir("a"))))
	assert.Equal(t, []path.Path{path.New("a/b.txt"), path.New("a")}, got)
}

func TestNewTree(t *testing.T) {
	t.Parallel()
	fsys, err := New(listing(
		memFile("a/b.txt", []byte("hello")),
		memDir("a/c"),
		memFile("z.txt", []byte("zee")),
	))
	require.NoError(t, err)

	ents, err := fsys.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())

	ents, err = fsys.ReadDir("a")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "b.txt", ents[0].Name())
	assert.Equal(t, "c", ents[1].Name())
	assert.True(t, ents[1].IsDir())

	b, err := fsys.ReadFile("a/b.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))

	st, err := fsys.Stat("a/c")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, iofs.ErrNotExist)

	_, err = fsys.Open("../x")
	require.ErrorIs(t, err, iofs.ErrInvalid)

	_, err = fsys.ReadFile("a")
	require.ErrorIs(t, err, iofs.ErrInvalid)

	_, err = fsys.ReadDir("z.txt")
	require.ErrorIs(t, err, iofs.ErrInvalid)
}

func TestNewLastWins(t *testing.T) {
	t.Parallel()
	fsys, err := New(listing(
		memFile("a.txt", []byte("old")),
		memFile("a.txt", []byte("new")),
	))
	require.NoError(t, err)
	b, err := fsys.ReadFile("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "new", string(b))
}

func TestNewFileDirConflict(t *testing.T) {
	t.Parallel()
	_, err := New(listing(memFile("a", []byte("x")), memDir("a")))
	require.ErrorIs(t, err, iofs.ErrExist)
}

func TestNewInvalidName(t *testing.T) {
	t.Parallel()
	_, err := New(listing(File{Name: path.New("../x"), Mode: 0o644}))
	require.ErrorIs(t, err, iofs.ErrInvalid)
}

func TestNewTestFS(t *testing.T) {
	t.Parallel()
	fsys, err := New(listing(
		memFile("a/b.txt", []byte("hello")),
		memFile("z.txt", []byte("zee")),
	))
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestNewWriteReadOnly(t *testing.T) {
	t.Parallel()
	fsys, err := New(listing(memFile("a.txt", []byte("x"))))
	require.NoError(t, err)
	err = path.New("a.txt").WriteFile(fsys, []byte("y"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestNewListingError(t *testing.T) {
	t.Parallel()
	_, err := New(func(yield func(File, error) bool) {
		yield(File{}, io.ErrUnexpectedEOF)
	})
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestNewDirMerge(t *testing.T) {
	t.Parallel()
	old := time.Unix(1, 0)
	late := time.Unix(2, 0)
	fsys, err := New(listing(
		File{Name: path.New("a"), Mode: iofs.ModeDir | 0o555, ModTime: old},
		File{Name: path.New("a"), Mode: iofs.ModeDir | 0o555, ModTime: late},
		memFile("a/b.txt", []byte("x")),
	))
	require.NoError(t, err)
	st, err := fsys.Stat("a")
	require.NoError(t, err)
	assert.Equal(t, late, st.ModTime())
}
