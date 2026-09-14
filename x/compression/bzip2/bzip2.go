// Package bzip2 is the bzip2 [github.com/lewtec/lewkit/x/compression.Codec].
package bzip2

import (
	stdbzip2 "compress/bzip2"
	"io"

	"github.com/lewtec/lewkit/x/compression"
)

// Codec is bzip2.
var Codec = codec{}

func init() { compression.Register(Codec) }

type codec struct{}

func (codec) Name() string { return "bzip2" }

func (codec) Extensions() []string { return []string{".bz2"} }

func (codec) Magic() [][]byte {
	return [][]byte{{'B', 'Z', 'h'}}
}

func (codec) Reader(r io.Reader) (io.Reader, error) {
	return stdbzip2.NewReader(r), nil
}
