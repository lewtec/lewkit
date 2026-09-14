package all_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/compression"
	"github.com/lewtec/lewkit/x/compression/all"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Parallel()
	all.Load()
	c, ok := compression.Detect("src.tar.gz", nil)
	require.True(t, ok)
	assert.Equal(t, "gzip", c.Name())

	c, ok = compression.Detect("a.tar.br", nil)
	require.True(t, ok)
	assert.Equal(t, "brotli", c.Name())

	c, ok = compression.ByMagic([]byte{0x04, 0x22, 0x4d, 0x18})
	require.True(t, ok)
	assert.Equal(t, "lz4", c.Name())
}
