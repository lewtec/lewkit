//go:build amd64

package avcodec

//go:noescape
func snappyCopySSE2(dst, src *byte, n int)
