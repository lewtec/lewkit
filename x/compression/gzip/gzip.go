// Package gzip is the gzip [github.com/lewtec/lewkit/x/compression.Codec].
package gzip

import (
	stdgzip "compress/gzip"
	"io"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is gzip.
var Codec compression.Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "gzip" }

func (codec) Extensions() []string {
	return []string{".gz", ".tgz", ".tar.gz"}
}

func (codec) Magic() [][]byte {
	return [][]byte{{0x1f, 0x8b}}
}

func (codec) Reader(r io.Reader) (io.ReadCloser, error) {
	return stdgzip.NewReader(r)
}
