//go:build !amd64

package avcodec

func addClip8SSE2(*byte, *int16, int) {}
