// Package compression is a process-wide registry of stream codecs
// keyed by file extension or magic prefix.
//
// Call [github.com/lewtec/lewkit/x/compression/all.Load] (or import that
// package) to fill the process-wide registry.
//
//	all.Load()
//	c, ok := compression.Detect("src.tar.gz", header)
//	r, err := c.Reader(src)
//
// [Detect] prefers a matching extension, then a magic prefix.
// Brotli has no reliable magic; it matches by extension only.
//
// [New] builds a private registry when the process-wide one is not wanted.
package compression
