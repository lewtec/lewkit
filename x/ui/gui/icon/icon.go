// Package icon paints one Lucide glyph as a gui node.
package icon

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/lewtec/lewkit/x/http/asset/lucide"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type key struct {
	name lucide.Name
	ink  gui.RGB
	px   int
}

var (
	mu    sync.Mutex
	cache = map[key]image.Image{}
)

// Node is one Lucide icon, size by size logical pixels, painted in ink.
// Layout rasters the SVG at size times the window density, so the stroke
// is drawn for that pixel square. A name this set does not know leaves the
// image empty.
func Node(name lucide.Name, ink gui.RGB, size float32) gui.Node {
	return &glyph{name: name, ink: ink, side: size}
}

type glyph struct {
	name   lucide.Name
	ink    gui.RGB
	side   float32
	pixels int
	image  *gui.Image
}

func (g *glyph) Layout(constraints gui.BoxConstraints) gui.Size {
	if g == nil {
		return gui.Size{}
	}
	px := iconPixels(g.side, constraints.Density())
	if g.image == nil || g.pixels != px {
		g.pixels = px
		g.image = &gui.Image{Src: paint(g.name, g.ink, px), Width: g.side, Height: g.side}
	}
	return g.image.Layout(constraints)
}

func (g *glyph) Paint(origin gui.Offset, clip gui.Rect, picture *gui.Picture) *ndarray.Tensor[float32] {
	if g == nil || g.image == nil {
		var blank gui.Image
		return blank.Paint(origin, clip, picture)
	}
	return g.image.Paint(origin, clip, picture)
}

func iconPixels(side, density float32) int {
	if density <= 0 {
		density = 1
	}
	n := int(math.Round(float64(side) * float64(density)))
	if n < 1 {
		return 1
	}
	return n
}

func paint(name lucide.Name, ink gui.RGB, px int) image.Image {
	id := key{name: name, ink: ink, px: px}
	mu.Lock()
	defer mu.Unlock()
	if img, ok := cache[id]; ok {
		return img
	}
	img, err := lucide.ImageSize(name, color.NRGBA{R: ink.Red, G: ink.Green, B: ink.Blue, A: ink.Alpha}, px)
	if err != nil {
		cache[id] = nil
		return nil
	}
	cache[id] = img
	return img
}
