package path

import (
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
	got, err := OpenFS(New("dir", "a.txt"), fsys, func(r io.Reader) (string, error) {
		b, err := io.ReadAll(r)
		return string(b), err
	})
	require.NoError(t, err)
	assert.Equal(t, "hi", got)
}

func TestOpenFSMissing(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{}
	_, err := OpenFS(New("nope"), fsys, func(io.Reader) (string, error) {
		t.Fatal("open must not run")
		return "", nil
	})
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestOpenFSClosesOnError(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a.txt": {Data: []byte("hi")}}
	_, err := OpenFS(New("a.txt"), fsys, func(io.Reader) (string, error) {
		return "", fs.ErrInvalid
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
}
