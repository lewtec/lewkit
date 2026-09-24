package image

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var white = image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255})

// Label draws s in white at (x, y) using [Face].
func Label(dst *image.RGBA, x, y int, s string) {
	Stamp{Dst: dst, X: x, Y: y, Text: s}.Draw()
}

// Stamp draws Text at (X, Y). Nil Face uses [Face]. Nil Src uses white.
type Stamp struct {
	Dst  *image.RGBA
	X, Y int
	Text string
	Face font.Face
	Src  image.Image
}

func (s Stamp) Draw() {
	if s.Dst == nil || s.Text == "" {
		return
	}
	src := s.Src
	if src == nil {
		src = white
	}
	d := &font.Drawer{
		Dst:  s.Dst,
		Src:  src,
		Face: Use(s.Face),
		Dot:  fixed.P(s.X, s.Y),
	}
	d.DrawString(s.Text)
}
