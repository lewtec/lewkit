//go:build !amd64

package avcodec

func snappyCopySSE2(*byte, *byte, int) {}
