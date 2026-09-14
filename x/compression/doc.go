// Package compression is a process-wide registry of stream codecs
// keyed by file extension or magic prefix.
//
// Import a codec package to register it, or import
// [github.com/lewtec/lewkit/x/compression/prelude] for the standard set.
//
//	import _ "github.com/lewtec/lewkit/x/compression/prelude"
//	c, ok := compression.Detect("src.tar.gz", header)
//	d := c.(compression.Decompressor)
//	r, err := d.Reader(src)
//
// [Detect] prefers a matching extension, then a magic prefix.
// A codec may also be a [Decompressor] and/or a [Compressor].
//
// [RoundTrip] is the codec test helper: compress, then decompress.
//
// [New] builds a private registry when the process-wide one is not wanted.
package compression
