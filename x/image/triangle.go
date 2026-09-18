package image

import (
	"image"
	"image/color"
	"image/draw"
)

// Triangle paints the vulkan-tutorial RGB triangle into dst.
// Top is red, bottom-right green, bottom-left blue. The rest is black.
func Triangle(dst *image.RGBA) {
	b := dst.Bounds()
	draw.Draw(dst, b, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	if b.Dx() < 1 || b.Dy() < 1 {
		return
	}
	ax, ay := ndc(0, -0.5, b)
	bx, by := ndc(0.5, 0.5, b)
	cx, cy := ndc(-0.5, 0.5, b)
	minX := max(b.Min.X, int(min(ax, bx, cx)))
	maxX := min(b.Max.X, int(max(ax, bx, cx))+1)
	minY := max(b.Min.Y, int(min(ay, by, cy)))
	maxY := min(b.Max.Y, int(max(ay, by, cy))+1)
	den := (by-cy)*(ax-cx) + (cx-bx)*(ay-cy)
	if den == 0 {
		return
	}
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			u := ((by-cy)*(px-cx) + (cx-bx)*(py-cy)) / den
			v := ((cy-ay)*(px-cx) + (ax-cx)*(py-cy)) / den
			wt := 1 - u - v
			if u < 0 || v < 0 || wt < 0 {
				continue
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(u * 255),
				G: uint8(v * 255),
				B: uint8(wt * 255),
				A: 255,
			})
		}
	}
}

func ndc(nx, ny float64, b image.Rectangle) (x, y float64) {
	return float64(b.Min.X) + (nx+1)*0.5*float64(b.Dx()), float64(b.Min.Y) + (ny+1)*0.5*float64(b.Dy())
}
