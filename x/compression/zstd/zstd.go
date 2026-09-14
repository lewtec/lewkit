// Package zstd is the Zstandard [github.com/lewtec/lewkit/x/compression.Codec].
package zstd

import (
	"io"

	"github.com/klauspost/compress/zstd"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is zstd.
var Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "zstd" }

func (codec) Extensions() []string { return []string{".zst", ".zstd"} }

func (codec) Magic() [][]byte {
	return [][]byte{{0x28, 0xb5, 0x2f, 0xfd}}
}

func (codec) Reader(r io.Reader) (io.ReadCloser, error) {
	d, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &closer{d}, nil
}

func (codec) Writer(w io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(w)
}

type closer struct{ *zstd.Decoder }

func (c *closer) Close() error {
	c.Decoder.Close()
	return nil
}
