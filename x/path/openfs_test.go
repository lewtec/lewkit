package path

import (
	"context"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenFS(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"dir/a.txt": {Data: []byte("hi")}}
	got, err := OpenFS(t.Context(), New("dir", "a.txt"), fsys, func(_ context.Context, r io.Reader) (string, error) {
		b, err := io.ReadAll(r)
		return string(b), err
	})
	require.NoError(t, err)
	assert.Equal(t, "hi", got)
}

func TestOpenFSMissing(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{}
	_, err := OpenFS(t.Context(), New("nope"), fsys, func(context.Context, io.Reader) (string, error) {
		t.Fatal("open must not run")
		return "", nil
	})
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestOpenFSCancel(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a.txt": {Data: []byte("hi")}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := OpenFS(ctx, New("a.txt"), fsys, func(context.Context, io.Reader) (string, error) {
		t.Fatal("open must not run")
		return "", nil
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestOpenFSClosesOnError(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a.txt": {Data: []byte("hi")}}
	_, err := OpenFS(t.Context(), New("a.txt"), fsys, func(context.Context, io.Reader) (string, error) {
		return "", fs.ErrInvalid
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
}
