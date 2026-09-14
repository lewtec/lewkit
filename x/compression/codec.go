package compression

import "io"

// Codec is one stream compression format.
type Codec interface {
	// Name is the short format name, such as "gzip".
	Name() string
	// Extensions are file suffixes, including the dot. Longer suffixes
	// win when several match (".tar.gz" before ".gz").
	Extensions() []string
	// Magic is leading byte prefixes of the compressed stream.
	// An empty list means the codec cannot be sniffed.
	Magic() [][]byte
	// Reader decompresses r. The caller closes the result.
	Reader(r io.Reader) (io.ReadCloser, error)
}
