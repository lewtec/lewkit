package fs

import (
	"context"
	iofs "io/fs"
	"testing"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyTree(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"a.txt":     {Data: []byte("hi")},
		"d/b.txt":   {Data: []byte("b")},
		"d/e/c.txt": {Data: []byte("c")},
		"empty":     {Mode: iofs.ModeDir},
	}
	root, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, root)
	require.NoError(t, path.New("out").MkdirAll(root, 0o777))
	dest, err := path.New("out").OpenRoot(root)
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)

	require.NoError(t, Copy(t.Context(), src, dest))

	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
	b, err = path.New("d", "e", "c.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("c"), b)
	ok, err := path.New("empty").IsDir(dest)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = path.New("a.txt").Exists(root)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCopyExist(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, path.New("a.txt").WriteFile(dest, []byte("old"), 0o644))
	err = Copy(t.Context(), fstest.MapFS{"a.txt": {Data: []byte("new")}}, dest)
	require.ErrorIs(t, err, iofs.ErrExist)
	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("old"), b)
}

func TestCopyCancel(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = Copy(ctx, fstest.MapFS{"a.txt": {Data: []byte("x")}}, dest)
	require.ErrorIs(t, err, context.Canceled)
}

func TestCopySymlink(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	src := fstest.MapFS{"link": {Mode: iofs.ModeSymlink, Data: []byte("a.txt")}}
	err = Copy(t.Context(), src, dest)
	require.ErrorIs(t, err, iofs.ErrInvalid)
}
