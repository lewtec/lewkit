package xz

import (
	"bytes"
	"io"
	"testing"

	stdxz "github.com/ulikunitz/xz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w, err := stdxz.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	r, err := Codec.Reader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	t.Cleanup(func() { r.Close() })
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), got)
}
