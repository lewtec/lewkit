package lucide

import (
	"image"
	"image/color"

	"github.com/lewtec/lewkit/x/image/convert/svg/render"
)

// Version is the Lucide release these bodies were taken from.
const Version = "1.57.0"

// SVG returns the icon drawn with stroke currentColor.
// The bool is false when name is not in this set.
func SVG(name Name) (string, bool) {
	body, ok := iconBody[name]
	return body, ok
}

// Image paints name in c. A nil color is black.
// The raster is 24 by 24, which is the Lucide grid.
func Image(name Name, c color.Color) (image.Image, error) {
	body, ok := iconBody[name]
	if !ok {
		return nil, ErrName
	}
	if c == nil {
		c = color.Black
	}
	painted, err := guiSVG(body, color.NRGBAModel.Convert(c).(color.NRGBA))
	if err != nil {
		return nil, err
	}
	return render.Decode([]byte(painted))
}
