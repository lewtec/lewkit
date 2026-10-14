package test

import (
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

// ArchiveTree checks a two-member archive: a/b.txt "hello" and z.txt "zee".
// Reading the directory "a" is left to the caller. Tar reports fs.ErrInvalid;
// zip reports a non-sentinel "is a directory" error.
func ArchiveTree(tb testing.TB, fsys fs.FS) {
	tb.Helper()
	ents, err := fs.ReadDir(fsys, ".")
	require.NoError(tb, err)
	require.Len(tb, ents, 2)
	require.Equal(tb, "a", ents[0].Name())
	require.True(tb, ents[0].IsDir())
	require.Equal(tb, "z.txt", ents[1].Name())
	require.False(tb, ents[1].IsDir())

	ents, err = fs.ReadDir(fsys, "a")
	require.NoError(tb, err)
	require.Len(tb, ents, 1)
	require.Equal(tb, "b.txt", ents[0].Name())

	b, err := fs.ReadFile(fsys, "a/b.txt")
	require.NoError(tb, err)
	require.Equal(tb, "hello", string(b))

	st, err := fs.Stat(fsys, "a")
	require.NoError(tb, err)
	require.True(tb, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(tb, err, fs.ErrNotExist)
	_, err = fsys.Open("../x")
	require.ErrorIs(tb, err, fs.ErrInvalid)
}

// ArchiveReadAt checks a.bin "hello world" via ReadAt at offset 6.
func ArchiveReadAt(tb testing.TB, fsys fs.FS) {
	tb.Helper()
	f, err := fsys.Open("a.bin")
	require.NoError(tb, err)
	CloseOnCleanup(tb, f)
	ra, ok := f.(io.ReaderAt)
	require.True(tb, ok)
	buf := make([]byte, 5)
	n, err := ra.ReadAt(buf, 6)
	require.NoError(tb, err)
	require.Equal(tb, 5, n)
	require.Equal(tb, "world", string(buf))
}
