package compression

import (
	"bytes"
	"io"
	"testing"

	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RoundTrip compresses p with c and decompresses the result.
// c must be a [Compressor] and a [Decompressor].
func RoundTrip(tb testing.TB, c Codec, p []byte) {
	tb.Helper()
	wri, ok := c.(Compressor)
	require.True(tb, ok, "%s: not a Compressor", c.Name())
	rdr, ok := c.(Decompressor)
	require.True(tb, ok, "%s: not a Decompressor", c.Name())
	var buf bytes.Buffer
	w, err := wri.Writer(&buf)
	require.NoError(tb, err)
	_, err = w.Write(p)
	require.NoError(tb, err)
	require.NoError(tb, w.Close())
	r, err := rdr.Reader(bytes.NewReader(buf.Bytes()))
	require.NoError(tb, err)
	test.CloseOnCleanup(tb, r)
	got, err := io.ReadAll(r)
	require.NoError(tb, err)
	assert.Equal(tb, p, got)
}
