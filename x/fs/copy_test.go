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

	require.NoError(t, Copy(t.Context(), dest, Walk(src, nil)))

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
	err = Copy(t.Context(), dest, Walk(fstest.MapFS{"a.txt": {Data: []byte("new")}}, nil))
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
	err = Copy(ctx, dest, Walk(fstest.MapFS{"a.txt": {Data: []byte("x")}}, nil))
	require.ErrorIs(t, err, context.Canceled)
}

func TestCopySymlink(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	src := fstest.MapFS{"link": {Mode: iofs.ModeSymlink, Data: []byte("a.txt")}}
	err = Copy(t.Context(), dest, Walk(src, nil))
	require.ErrorIs(t, err, iofs.ErrInvalid)
}

func keepGlob(pattern string) Keep {
	return func(p path.Path, dir bool) bool {
		if dir {
			return true
		}
		ok, err := p.MatchGlob(pattern)
		return err == nil && ok
	}
}

func TestCopyKeep(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"a.txt":   {Data: []byte("hi")},
		"d/b.go":  {Data: []byte("pkg")},
		"d/c.txt": {Data: []byte("c")},
		"empty":   {Mode: iofs.ModeDir},
	}
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, Copy(t.Context(), dest, Walk(src, keepGlob("**/*.txt"))))
	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
	b, err = path.New("d", "c.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("c"), b)
	ok, err := path.New("d", "b.go").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = path.New("empty").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCopyFiles(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, Copy(t.Context(), dest, listing(
		memDir("d"),
		memFile("d/b.txt", []byte("b")),
		memFile("a.txt", []byte("hi")),
	)))
	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
	b, err = path.New("d", "b.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("b"), b)
}

func TestCopyFilesKeep(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, Copy(t.Context(), dest, Filter(listing(
		memDir("d"),
		memFile("d/b.go", []byte("pkg")),
		memFile("d/c.txt", []byte("c")),
		memDir("empty"),
	), keepGlob("**/*.txt"))))
	b, err := path.New("d", "c.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("c"), b)
	ok, err := path.New("d", "b.go").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = path.New("empty").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
}

func pruneSkip(p path.Path, dir bool) bool {
	return !(dir && p.String() == "skip")
}

func TestCopyPrune(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"a.txt":      {Data: []byte("a")},
		"skip/x.txt": {Data: []byte("x")},
	}
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, Copy(t.Context(), dest, Walk(src, pruneSkip)))
	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("a"), b)
	ok, err := path.New("skip", "x.txt").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCopyFilesPrune(t *testing.T) {
	t.Parallel()
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, Copy(t.Context(), dest, Filter(listing(
		memFile("a.txt", []byte("a")),
		memFile("skip/x.txt", []byte("x")),
	), pruneSkip)))
	b, err := path.New("a.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("a"), b)
	ok, err := path.New("skip", "x.txt").Exists(dest)
	require.NoError(t, err)
	assert.False(t, ok)
}
