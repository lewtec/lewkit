// Package brotli is the brotli [github.com/lewtec/lewkit/x/compression.Codec].
package brotli

import (
	"bytes"
	"io"

	"github.com/andybalholm/brotli"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is brotli.
var Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "brotli" }

func (codec) Extensions() []string { return []string{".br"} }

func (codec) Magic() [][]byte {
	return [][]byte{{0xce, 0xb2, 0xcf, 0x81}}
}

func (codec) Reader(r io.Reader) (io.Reader, error) {
	var hdr [4]byte
	n, err := io.ReadFull(r, hdr[:])
	if err == nil && bytes.Equal(hdr[:], []byte{0xce, 0xb2, 0xcf, 0x81}) {
		return brotli.NewReader(r), nil
	}
	return brotli.NewReader(io.MultiReader(bytes.NewReader(hdr[:n]), r)), nil
}

func (codec) Writer(w io.Writer) (io.WriteCloser, error) {
	return brotli.NewWriter(w), nil
}
