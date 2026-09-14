package lz4

import (
	"bytes"
	"io"
	"testing"

	stdlz4 "github.com/pierrec/lz4/v4"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stdlz4.NewWriter(&buf)
	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	r, err := Codec.Reader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	t.Cleanup(func() { r.Close() })
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), got)
}
