//go:build amd64

package avcodec

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

//go:noescape
func gsmMult40SSE2(dst *int16, src *int16, gain int16)

func gsmMult40Asm(dst, src []int16, gain int16) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE41 || len(dst) < 40 || len(src) < 40 {
		return false
	}
	gsmMult40SSE2(&dst[0], &src[0], gain)
	return true
}
