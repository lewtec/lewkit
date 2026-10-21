//go:build amd64

package avcodec

//go:noescape
func addClip8SSE2(dst *byte, residual *int16, n int)
