//go:build !amd64

package avcodec

func hevcCopy16SSE2(*byte, *byte) {}
