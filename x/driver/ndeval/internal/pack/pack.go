// Package pack widens host bytes into the GPU word layout the evaluators share.
package pack

import "github.com/lewtec/lewkit/x/ndarray"

// Bytes is the GPU buffer size for n host bytes of dtype.
// A U8 value occupies one little-endian uint32.
func Bytes(dtype ndarray.DType, n int) int {
	if dtype == ndarray.U8 {
		return n * 4
	}
	return n
}

// Widen copies a U8 host buffer into little-endian uint32 words.
// Any other dtype returns raw and leaves staging alone.
// The returned data is the slice to store. next is the staging buffer to keep.
func Widen(dtype ndarray.DType, raw, staging []byte) (data, next []byte) {
	if dtype != ndarray.U8 {
		return raw, staging
	}
	need := len(raw) * 4
	if cap(staging) < need {
		staging = make([]byte, need)
	} else {
		staging = staging[:need]
		clear(staging)
	}
	for j, v := range raw {
		staging[j*4] = v
	}
	return staging, staging
}
