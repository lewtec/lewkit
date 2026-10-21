//go:build amd64

package avcodec

//go:noescape
func hevcCopy16SSE2(dst, src *byte)
