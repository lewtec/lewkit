package test

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnlyReader(t *testing.T) {
	r := OnlyReader{strings.NewReader("x")}
	_, ok := any(r).(io.ReaderAt)
	require.False(t, ok)
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "x", string(b))
}

func TestReaderOnlyFS(t *testing.T) {
	const name = "a.tar"
	fsys := ReaderOnlyFS(name, strings.NewReader("x"))
	f, err := fsys.Open(name)
	require.NoError(t, err)
	_, ok := f.(readerOnlyFile)
	require.True(t, ok)
	_, ok = f.(io.ReaderAt)
	require.False(t, ok)
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, "x", string(b))
	_, err = f.Stat()
	require.ErrorIs(t, err, fs.ErrInvalid)
	require.NoError(t, f.Close())
	_, err = fsys.Open("other")
	require.ErrorIs(t, err, fs.ErrNotExist)
}
