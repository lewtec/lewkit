//go:build amd64

package swresample

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

//go:noescape
func s16ToF32AsmRaw(dst *float32, src *int16, n int)

//go:noescape
func f32ToS16AsmRaw(dst *int16, src *float32, n int)

func s16ToF32Asm(dst []float32, src []int16) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE2 || len(src) < 8 || len(dst) < 8 {
		return false
	}
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	s16ToF32AsmRaw(&dst[0], &src[0], n&^7)
	return true
}

func f32ToS16Asm(dst []int16, src []float32) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE2 || len(src) < 8 || len(dst) < 8 {
		return false
	}
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	f32ToS16AsmRaw(&dst[0], &src[0], n&^7)
	return true
}
