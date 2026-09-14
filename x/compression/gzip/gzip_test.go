package gzip

import (
	"bytes"
	"io"
	"testing"

	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w, err := Codec.Writer(&buf)
	require.NoError(t, err)
	_, err = w.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	r, err := Codec.Reader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	if c, ok := r.(io.Closer); ok {
		test.CloseOnCleanup(t, c)
	}
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), got)
}
