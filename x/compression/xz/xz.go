// Package xz is the xz [github.com/lewtec/lewkit/x/compression.Codec].
package xz

import (
	"io"

	stdxz "github.com/ulikunitz/xz"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is xz.
var Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "xz" }

func (codec) Extensions() []string { return []string{".xz"} }

func (codec) Magic() [][]byte {
	return [][]byte{{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}}
}

func (codec) Reader(r io.Reader) (io.ReadCloser, error) {
	zr, err := stdxz.NewReader(r)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(zr), nil
}

func (codec) Writer(w io.Writer) (io.WriteCloser, error) {
	return stdxz.NewWriter(w)
}
