package avcodec

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

// snappyCopyFast copies length bytes that do not overlap. The scalar loop
// handles overlapping back-references.
func snappyCopyFast(dst, src []byte) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE2 || len(dst) < 16 || len(src) < 16 {
		return false
	}
	n := len(dst) &^ 15
	if n > len(src) {
		n = len(src) &^ 15
	}
	if n == 0 {
		return false
	}
	snappyCopySSE2(&dst[0], &src[0], n)
	copy(dst[n:], src[n:])
	return true
}
