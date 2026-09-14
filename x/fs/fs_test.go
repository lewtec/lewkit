package fs

import (
	"errors"
	"io"
	iofs "io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

func TestReaderAt(t *testing.T) {
	t.Parallel()
	_, err := ReaderAt("open", onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedReadAt)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	ra, err := ReaderAt("open", strings.NewReader("x"))
	require.NoError(t, err)
	assert.NotNil(t, ra)
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

func TestSize(t *testing.T) {
	t.Parallel()
	n, err := Size("open", strings.NewReader("hello"))
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)

	_, err = Size("open", onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedSize)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	_, err = Size("open", atOnly{b: []byte("xy")})
	require.ErrorIs(t, err, ErrNeedSize)
}
