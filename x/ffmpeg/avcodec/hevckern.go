package avcodec

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

// hevcCopy16 copies 16 bytes. The PCM packer uses it for each luma row.
func hevcCopy16(dst, src []byte) {
	if len(dst) < 16 || len(src) < 16 {
		return
	}
	if hevcCopy16Asm(dst, src) {
		return
	}
	copy(dst[:16], src[:16])
}

func hevcCopy16Asm(dst, src []byte) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE2 {
		return false
	}
	hevcCopy16SSE2(&dst[0], &src[0])
	return true
}
