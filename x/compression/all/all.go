// Package all loads the standard stream codecs into the
// process-wide [github.com/lewtec/lewkit/x/compression] registry.
package all

import (
	"github.com/lewtec/lewkit/x/compression"
	"github.com/lewtec/lewkit/x/compression/brotli"
	"github.com/lewtec/lewkit/x/compression/bzip2"
	"github.com/lewtec/lewkit/x/compression/gzip"
	"github.com/lewtec/lewkit/x/compression/lz4"
	"github.com/lewtec/lewkit/x/compression/xz"
	"github.com/lewtec/lewkit/x/compression/zstd"
)

// Load registers the standard codecs. Safe to call more than once.
func Load() {
	compression.Register(
		gzip.Codec,
		brotli.Codec,
		lz4.Codec,
		zstd.Codec,
		xz.Codec,
		bzip2.Codec,
	)
}

func init() { Load() }
