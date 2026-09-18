package image

import (
	"image"
	"math"
)

// Uncentered vulkan-tutorial verts, then shifted so the centroid is the origin.
const (
	triAX, triAY = 0, -2.0 / 3.0
	triBX, triBY = 0.5, 1.0 / 3.0
	triCX, triCY = -0.5, 1.0 / 3.0
)

// Triangle paints the vulkan-tutorial RGB triangle into dst.
// Top is red, bottom-right green, bottom-left blue. The rest is black.
func Triangle(dst *image.RGBA) {
	TriangleTurn(dst, 0)
}

// TriangleTurn is Triangle rotated by turn revolutions around the window center.
func TriangleTurn(dst *image.RGBA, turn float64) {
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return
	}
	clearBlack(dst)
	s, c := math.Sincos(turn * 2 * math.Pi)
	scale := float64(min(w, h)) / 2
	cx := float64(b.Min.X) + float64(w)/2
	cy := float64(b.Min.Y) + float64(h)/2
	ax, ay := px(triAX, triAY, s, c, scale, cx, cy)
	bx, by := px(triBX, triBY, s, c, scale, cx, cy)
	cxp, cyp := px(triCX, triCY, s, c, scale, cx, cy)
	minX := max(b.Min.X, int(min(ax, bx, cxp)))
	maxX := min(b.Max.X, int(max(ax, bx, cxp))+1)
	minY := max(b.Min.Y, int(min(ay, by, cyp)))
	maxY := min(b.Max.Y, int(max(ay, by, cyp))+1)
	den := (by-cyp)*(ax-cxp) + (cxp-bx)*(ay-cyp)
	if den == 0 {
		return
	}
	pix := dst.Pix
	for y := minY; y < maxY; y++ {
		off := dst.PixOffset(minX, y)
		for x := minX; x < maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			u := ((by-cyp)*(px-cxp) + (cxp-bx)*(py-cyp)) / den
			v := ((cyp-ay)*(px-cxp) + (ax-cxp)*(py-cyp)) / den
			wt := 1 - u - v
			if u >= 0 && v >= 0 && wt >= 0 {
				pix[off] = uint8(u * 255)
				pix[off+1] = uint8(v * 255)
				pix[off+2] = uint8(wt * 255)
				pix[off+3] = 255
			}
			off += 4
		}
	}
}

func clearBlack(dst *image.RGBA) {
	pix := dst.Pix
	for i := 0; i < len(pix); i += 4 {
		pix[i] = 0
		pix[i+1] = 0
		pix[i+2] = 0
		pix[i+3] = 255
	}
}

func px(x, y, s, c, scale, ox, oy float64) (float64, float64) {
	return ox + (x*c-y*s)*scale, oy + (x*s+y*c)*scale
}
