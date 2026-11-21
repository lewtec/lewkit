package image

import (
	"image"
	stdraw "image/draw"
)

func CenterSquare(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := min(h, w)
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	rect := image.Rect(x0, y0, x0+side, y0+side)
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if s, ok := src.(subImager); ok {
		return s.SubImage(rect)
	}
	out := image.NewRGBA(image.Rect(0, 0, side, side))
	stdraw.Draw(out, out.Bounds(), src, rect.Min, stdraw.Src)
	return out
}
