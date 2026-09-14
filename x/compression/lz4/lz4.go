// Package lz4 is the LZ4 frame [github.com/lewtec/lewkit/x/compression.Codec].
package lz4

import (
	"io"

	stdlz4 "github.com/pierrec/lz4/v4"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is LZ4 (frame format).
var Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "lz4" }

func (codec) Extensions() []string { return []string{".lz4"} }

func (codec) Magic() [][]byte {
	return [][]byte{{0x04, 0x22, 0x4d, 0x18}}
}

func (codec) Reader(r io.Reader) (io.Reader, error) {
	return stdlz4.NewReader(r), nil
}

func (codec) Writer(w io.Writer) (io.WriteCloser, error) {
	return stdlz4.NewWriter(w), nil
}
