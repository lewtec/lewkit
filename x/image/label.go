package image

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var white = image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255})

// Label draws s in white at (x, y) using the built-in 7×13 face.
func Label(dst *image.RGBA, x, y int, s string) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  white,
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
