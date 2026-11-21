package brotli

import (
	"testing"

	"github.com/lewtec/lewkit/x/compression"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	compression.RoundTrip(t, Codec, []byte("hello"))
}
