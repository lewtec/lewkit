package avcodec

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

// h264IDCTAdd applies the H.264 4x4 inverse residual transform and adds it
// to a byte block. block is cleared. The store uses addClip8.
func h264IDCTAdd(dst []byte, stride int, block *[16]int16) {
	block[0] += 32
	var col [16]int16
	for i := 0; i < 4; i++ {
		b0 := int32(block[i])
		b1 := int32(block[i+4])
		b2 := int32(block[i+8])
		b3 := int32(block[i+12])
		z0 := b0 + b2
		z1 := b0 - b2
		z2 := (b1 >> 1) - b3
		z3 := b1 + (b3 >> 1)
		col[i] = int16(z0 + z3)
		col[i+4] = int16(z1 + z2)
		col[i+8] = int16(z1 - z2)
		col[i+12] = int16(z0 - z3)
	}
	var rows [4][8]int16
	for i := 0; i < 4; i++ {
		b0 := int32(col[0+4*i])
		b1 := int32(col[1+4*i])
		b2 := int32(col[2+4*i])
		b3 := int32(col[3+4*i])
		z0 := b0 + b2
		z1 := b0 - b2
		z2 := (b1 >> 1) - b3
		z3 := b1 + (b3 >> 1)
		rows[i][0] = int16((z0 + z3) >> 6)
		rows[i][1] = int16((z1 + z2) >> 6)
		rows[i][2] = int16((z1 - z2) >> 6)
		rows[i][3] = int16((z0 - z3) >> 6)
	}
	for i := 0; i < 4; i++ {
		addClip8(dst[i*stride:i*stride+4], rows[i][:4])
	}
	*block = [16]int16{}
}

// addClip8 adds residuals onto bytes and saturates to 0..255.
func addClip8(dst []byte, residual []int16) {
	n := len(dst)
	if len(residual) < n {
		n = len(residual)
	}
	if n >= 8 && addClip8Asm(dst, residual) {
		for i := n &^ 7; i < n; i++ {
			dst[i] = clip8(int(dst[i]) + int(residual[i]))
		}
		return
	}
	for i := 0; i < n; i++ {
		dst[i] = clip8(int(dst[i]) + int(residual[i]))
	}
}

func clip8(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

func addClip8Asm(dst []byte, residual []int16) bool {
	if avutil.Scalar() || !avutil.HostCPU().SSE2 || len(dst) < 8 || len(residual) < 8 {
		return false
	}
	addClip8SSE2(&dst[0], &residual[0], len(dst)&^7)
	return true
}
