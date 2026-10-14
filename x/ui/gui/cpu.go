package gui

import (
	"image"
	"math"
)

// composite paints fills over under, then applies ink the way the fused
// kernel does. under is tightly packed RGBA of dst's size, or nil for
// opaque black. A non-zero ink channel replaces that channel.
func composite(dst *image.RGBA, fills []Draw, under []byte, ink *image.RGBA) {
	if dst == nil {
		return
	}
	width, height := dst.Rect.Dx(), dst.Rect.Dy()
	if width < 1 || height < 1 {
		return
	}
	useInk := ink != nil && ink.Rect.Dx() == width && ink.Rect.Dy() == height
	for y := 0; y < height; y++ {
		py := float32(y) + 0.5
		for x := 0; x < width; x++ {
			px := float32(x) + 0.5
			r, g, b, a := float32(0), float32(0), float32(0), float32(255)
			if under != nil {
				u := (y*width + x) * 4
				if u+3 < len(under) {
					r = float32(under[u])
					g = float32(under[u+1])
					b = float32(under[u+2])
					a = float32(under[u+3])
				}
			}
			for i := range fills {
				cover := coverageAt(px, py, fills[i])
				if cover == 0 {
					continue
				}
				alpha := cover * (fills[i].Alpha * (1.0 / 255))
				r += alpha * (fills[i].Red - r)
				g += alpha * (fills[i].Green - g)
				b += alpha * (fills[i].Blue - b)
				a += alpha * (255 - a)
			}
			if useInk {
				ip := ink.PixOffset(x, y)
				if ip+3 < len(ink.Pix) {
					if v := ink.Pix[ip]; v != 0 {
						r = float32(v)
					}
					if v := ink.Pix[ip+1]; v != 0 {
						g = float32(v)
					}
					if v := ink.Pix[ip+2]; v != 0 {
						b = float32(v)
					}
					if v := ink.Pix[ip+3]; v != 0 {
						a = float32(v)
					}
				}
			}
			off := dst.PixOffset(dst.Rect.Min.X+x, dst.Rect.Min.Y+y)
			dst.Pix[off] = quant8(r)
			dst.Pix[off+1] = quant8(g)
			dst.Pix[off+2] = quant8(b)
			dst.Pix[off+3] = quant8(a)
		}
	}
}

// coverageAt is the kernel's rounded-rect mask at a pixel center.
// One means the fill covers that sample. The clip is off when ClipWidth
// is below half a pixel.
func coverageAt(px, py float32, fill Draw) float32 {
	const half float32 = 0.5
	localX := px - fill.X + half - fill.Width*half
	localY := py - fill.Y + half - fill.Height*half
	halfW := fill.Width * half
	halfH := fill.Height * half
	radius := fill.Radius
	if radius > halfW {
		radius = halfW
	}
	if radius > halfH {
		radius = halfH
	}
	ax, ay := localX, localY
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	cornerX := ax - halfW + radius
	cornerY := ay - halfH + radius
	outsideX, outsideY := cornerX, cornerY
	if outsideX < 0 {
		outsideX = 0
	}
	if outsideY < 0 {
		outsideY = 0
	}
	cornerMax := cornerX
	if cornerY > cornerMax {
		cornerMax = cornerY
	}
	inside := float32(0)
	if cornerMax < 0 {
		inside = cornerMax
	}
	dist := inside + float32(math.Sqrt(float64(outsideX*outsideX+outsideY*outsideY))) - radius
	if dist >= half {
		return 0
	}
	if fill.ClipWidth >= half {
		if px < fill.ClipX || px >= fill.ClipX+fill.ClipWidth || py < fill.ClipY || py >= fill.ClipY+fill.ClipHeight {
			return 0
		}
	}
	return 1
}

func quant8(v float32) uint8 {
	n := int32(v)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}
