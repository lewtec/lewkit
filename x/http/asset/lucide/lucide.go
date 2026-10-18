package lucide

import (
	"image"
	"image/color"

	"github.com/lewtec/lewkit/x/image/convert/svg/render"
)

// SVG returns the icon drawn with stroke currentColor.
// The bool is false when name is not in this set.
func SVG(name Name) (string, bool) {
	body, ok := iconBody[name]
	return body, ok
}

// Image paints name in c on the Lucide grid, 24 by 24.
// A nil color is black.
func Image(name Name, c color.Color) (image.Image, error) {
	return imageSize(name, c, 0)
}

// ImageSize paints name in c on a square of side pixels.
// The viewBox stays the Lucide grid, so the stroke scales with side.
// A side below 1 uses the grid. A nil color is black.
func ImageSize(name Name, c color.Color, side int) (image.Image, error) {
	return imageSize(name, c, side)
}

func imageSize(name Name, c color.Color, side int) (image.Image, error) {
	body, ok := iconBody[name]
	if !ok {
		return nil, ErrName
	}
	if c == nil {
		c = color.Black
	}
	painted, err := guiSVG(body, color.NRGBAModel.Convert(c).(color.NRGBA), side)
	if err != nil {
		return nil, err
	}
	return render.Decode([]byte(painted))
}
