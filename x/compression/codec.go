package compression

import "io"

// Codec identifies a stream format.
type Codec interface {
	// Name is the short format name, such as "gzip".
	Name() string
	// Extensions are file suffixes, including the dot.
	// Longer suffixes win when several match.
	Extensions() []string
	// Magic is leading byte prefixes of the compressed stream.
	// An empty list means the codec cannot be sniffed.
	Magic() [][]byte
}

// Decompressor is an optional reader side of a [Codec].
// Reader always returns a closer. Use [io.NopCloser] when the
// underlying stream has no Close.
type Decompressor interface {
	Reader(r io.Reader) (io.ReadCloser, error)
}

// Compressor is an optional writer side of a [Codec].
type Compressor interface {
	Writer(w io.Writer) (io.WriteCloser, error)
}
