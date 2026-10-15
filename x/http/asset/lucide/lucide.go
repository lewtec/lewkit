package lucide

import (
	"image"
	"image/color"
	"sort"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/image/convert/svg/render"
)

// Version is the Lucide release these bodies were taken from.
const Version = "1.57.0"

// ErrNotFound is an icon name that is not in this set.
var ErrNotFound = errs.Wrap(errs.ErrNotFound, "unknown icon")

// SVG returns the icon drawn with stroke currentColor.
// The bool is false when name is not in this set.
func SVG(name string) (string, bool) {
	body, ok := iconBody[name]
	return body, ok
}

// Names lists every icon, in lexicographic order.
func Names() []string {
	names := make([]string, 0, len(iconBody))
	for name := range iconBody {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Image paints name in c. A nil color is black.
// The raster is 24 by 24, which is the Lucide grid.
func Image(name string, c color.Color) (image.Image, error) {
	body, ok := iconBody[name]
	if !ok {
		return nil, ErrNotFound
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
