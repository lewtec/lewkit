// Package brotli is the brotli [github.com/lewtec/lewkit/x/compression.Codec].
//
// Brotli has no reliable magic prefix. Detect it by extension only.
package brotli

import (
	"io"

	"github.com/andybalholm/brotli"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is brotli.
var Codec compression.Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "brotli" }

func (codec) Extensions() []string {
	return []string{".br", ".tar.br"}
}

func (codec) Magic() [][]byte { return nil }

func (codec) Reader(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(brotli.NewReader(r)), nil
}
