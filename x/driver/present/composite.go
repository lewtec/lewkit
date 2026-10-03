package present

import (
	"encoding/binary"
	"math"
)

// Composite is the CPU picture of [Screen.Draw].
// The clear is opaque black. Under, each fill, and ink are premultiplied
// and blended with source-one, destination one-minus-source-alpha.
// The result is tightly packed RGBA8 in that framebuffer, top to bottom.
// A short under or ink buffer is skipped. instances must be a multiple of
// [InstanceStride].
func Composite(instances, under, ink []byte, width, height int) ([]byte, error) {
	if width < 1 || height < 1 {
		return nil, ErrSize
	}
	if len(instances)%InstanceStride != 0 {
		return nil, ErrSize
	}
	dst := make([]byte, width*height*4)
	for i := 0; i < len(dst); i += 4 {
		dst[i+3] = 255
	}
	need := width * height * 4
	if len(under) >= need {
		blendImage(dst, under, width, height)
	}
	for off := 0; off < len(instances); off += InstanceStride {
		blendFill(dst, instances[off:off+InstanceStride], width, height)
	}
	if len(ink) >= need {
		blendImage(dst, ink, width, height)
	}
	return dst, nil
}

func blendImage(dst, src []byte, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4
			word := binary.LittleEndian.Uint32(src[i : i+4])
			r := float32(word&255) / 255
			g := float32((word>>8)&255) / 255
			b := float32((word>>16)&255) / 255
			a := float32(word>>24) / 255
			if r == 0 && g == 0 && b == 0 && a == 0 {
				continue
			}
			if a == 0 {
				a = 1
			}
			blend(dst[i:i+4], r*a, g*a, b*a, a)
		}
	}
}

func blendFill(dst, inst []byte, width, height int) {
	val := func(i int) float32 {
		return math.Float32frombits(binary.LittleEndian.Uint32(inst[i*4 : (i+1)*4]))
	}
	boxX, boxY, boxW, boxH := val(0), val(1), val(2), val(3)
	red, green, blue, alpha := val(4), val(5), val(6), val(7)
	radius := val(8)
	clipX, clipY, clipW, clipH := val(9), val(10), val(11), val(12)
	if boxW <= 0 || boxH <= 0 {
		return
	}
	for y := 0; y < height; y++ {
		py := float32(y) + 0.5
		for x := 0; x < width; x++ {
			px := float32(x) + 0.5
			if cover(px, py, boxX, boxY, boxW, boxH, radius, clipX, clipY, clipW, clipH) == 0 {
				continue
			}
			a := alpha / 255
			sr := red / 255 * a
			sg := green / 255 * a
			sb := blue / 255 * a
			i := (y*width + x) * 4
			blend(dst[i:i+4], sr, sg, sb, a)
		}
	}
}

func cover(px, py, boxX, boxY, boxW, boxH, radius, clipX, clipY, clipW, clipH float32) float32 {
	const half = 0.5
	localX := px - boxX + half - boxW*half
	localY := py - boxY + half - boxH*half
	halfW := boxW * half
	halfH := boxH * half
	if radius > halfW {
		radius = halfW
	}
	if radius > halfH {
		radius = halfH
	}
	cx := abs(localX) - halfW + radius
	cy := abs(localY) - halfH + radius
	ox := cx
	if ox < 0 {
		ox = 0
	}
	oy := cy
	if oy < 0 {
		oy = 0
	}
	cornerMax := cx
	if cy > cornerMax {
		cornerMax = cy
	}
	inside := float32(0)
	if cornerMax < 0 {
		inside = cornerMax
	}
	dist := inside + float32(math.Sqrt(float64(ox*ox+oy*oy))) - radius
	covered := float32(0)
	if dist < half {
		covered = 1
	}
	if clipW >= half {
		if px < clipX || px >= clipX+clipW || py < clipY || py >= clipY+clipH {
			return 0
		}
	}
	return covered
}

func blend(dst []byte, sr, sg, sb, sa float32) {
	dr := float32(dst[0]) / 255
	dg := float32(dst[1]) / 255
	db := float32(dst[2]) / 255
	da := float32(dst[3]) / 255
	keep := 1 - sa
	dst[0] = quant(sr + dr*keep)
	dst[1] = quant(sg + dg*keep)
	dst[2] = quant(sb + db*keep)
	dst[3] = quant(sa + da*keep)
}

func quant(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
