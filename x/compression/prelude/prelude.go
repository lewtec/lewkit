// Package prelude registers the standard stream codecs via blank imports.
package prelude

import (
	_ "github.com/lewtec/lewkit/x/compression/brotli"
	_ "github.com/lewtec/lewkit/x/compression/bzip2"
	_ "github.com/lewtec/lewkit/x/compression/gzip"
	_ "github.com/lewtec/lewkit/x/compression/lz4"
	_ "github.com/lewtec/lewkit/x/compression/xz"
	_ "github.com/lewtec/lewkit/x/compression/zstd"
)
