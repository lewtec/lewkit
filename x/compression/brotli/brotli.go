// Package brotli is the brotli [github.com/lewtec/lewkit/x/compression.Codec].
package brotli

import (
	"io"

	"github.com/andybalholm/brotli"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is brotli.
var Codec = codec{}

func init() { compression.MustRegister(Codec) }

type codec struct{}

func (codec) Name() string { return "brotli" }

func (codec) Extensions() []string { return []string{".br"} }

// RFC 7932 is a raw bitstream. There is no official magic.
func (codec) Magic() [][]byte { return nil }

func (codec) Reader(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(brotli.NewReader(r)), nil
}

func (codec) Writer(w io.Writer) (io.WriteCloser, error) {
	return brotli.NewWriter(w), nil
}
