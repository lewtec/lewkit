package udf

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestInvalidVolume(t *testing.T) {
	t.Parallel()
	_, err := Open(bytes.NewReader(make([]byte, 4096)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNeedReadAt))
}

func TestTree(t *testing.T) {
	t.Parallel()
	root := newDir(".")
	require.NoError(t, root.add("a/b.txt", nil, false))
	require.NoError(t, root.add("a/c", nil, true))
	require.NoError(t, root.add("z.txt", nil, false))
	fsys := &FS{root: root}

	ents, err := fsys.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())
	assert.False(t, ents[1].IsDir())

	ents, err = fsys.ReadDir("a")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "b.txt", ents[0].Name())
	assert.Equal(t, "c", ents[1].Name())
	assert.True(t, ents[1].IsDir())

	st, err := fsys.Stat("a/c")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.Open("../x")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadFile("a")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadDir("z.txt")
	require.ErrorIs(t, err, fs.ErrInvalid)

	f, err := fsys.Open("z.txt")
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.Nil(t, f)
}

func TestPathReadDir(t *testing.T) {
	t.Parallel()
	root := newDir(".")
	require.NoError(t, root.add("sources/install.wim", nil, false))
	fsys := &FS{root: root}
	ents, err := path.New("sources").ReadDir(fsys)
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "install.wim", ents[0].Name())
}

func TestDuplicateFile(t *testing.T) {
	t.Parallel()
	root := newDir(".")
	require.NoError(t, root.add("a.txt", nil, false))
	err := root.add("a.txt", nil, false)
	require.ErrorIs(t, err, fs.ErrExist)
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	root := newDir(".")
	require.NoError(t, root.add("a.txt", nil, false))
	fsys := &FS{root: root}
	err := path.New("a.txt").WriteFile(fsys, []byte("x"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestAddDirTwice(t *testing.T) {
	t.Parallel()
	root := newDir(".")
	require.NoError(t, root.add("a", nil, true))
	require.NoError(t, root.add("a", nil, true))
	require.NoError(t, root.add("a/b.txt", nil, false))
}
