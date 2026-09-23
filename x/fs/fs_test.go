package fs

import (
	"errors"
	"io"
	iofs "io/fs"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderAt(t *testing.T) {
	t.Parallel()
	_, err := ReaderAt("open", test.OnlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedReadAt)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	ra, err := ReaderAt("open", strings.NewReader("x"))
	require.NoError(t, err)
	assert.NotNil(t, ra)
}

func TestCheckName(t *testing.T) {
	t.Parallel()
	require.NoError(t, CheckName("open", "."))
	require.NoError(t, CheckName("open", ""))
	require.NoError(t, CheckName("open", "a/b"))

	err := CheckName("open", "../x")
	require.ErrorIs(t, err, iofs.ErrInvalid)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
	assert.Equal(t, "../x", pe.Path)
}

type atOnly struct{ b []byte }

func (a atOnly) Read(p []byte) (int, error) {
	n := copy(p, a.b)
	if n < len(a.b) {
		return n, nil
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, io.EOF
}

func (a atOnly) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(a.b)) {
		return 0, io.EOF
	}
	n := copy(p, a.b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

type nameEnt string

func (e nameEnt) Name() string               { return string(e) }
func (nameEnt) IsDir() bool                  { return false }
func (nameEnt) Type() iofs.FileMode          { return 0 }
func (nameEnt) Info() (iofs.FileInfo, error) { return nil, iofs.ErrInvalid }

func TestDirEntries(t *testing.T) {
	t.Parallel()
	ents := []iofs.DirEntry{nameEnt("a"), nameEnt("b"), nameEnt("c")}

	off := 0
	got, err := DirEntries(ents, &off, 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0].Name())
	assert.Equal(t, 1, off)

	got, err = DirEntries(ents, &off, 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "b", got[0].Name())
	assert.Equal(t, "c", got[1].Name())
	assert.Equal(t, 3, off)

	got, err = DirEntries(ents, &off, 1)
	require.ErrorIs(t, err, io.EOF)
	assert.Nil(t, got)

	got, err = DirEntries(ents, &off, 0)
	require.NoError(t, err)
	assert.Nil(t, got)

	off = 1
	got, err = DirEntries(ents, &off, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "b", got[0].Name())
	assert.Equal(t, "c", got[1].Name())
	assert.Equal(t, 3, off)
}

func TestSize(t *testing.T) {
	t.Parallel()
	n, err := Size("open", strings.NewReader("hello"))
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)

	_, err = Size("open", test.OnlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedSize)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	_, err = Size("open", atOnly{b: []byte("xy")})
	require.ErrorIs(t, err, ErrNeedSize)
}
