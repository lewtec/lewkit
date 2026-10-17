// Package icon paints one Lucide glyph for an example.
package icon

import (
	"image"
	"image/color"
	"sync"

	"github.com/lewtec/lewkit/x/http/asset/lucide"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type key struct {
	name lucide.Name
	ink  gui.RGB
}

var (
	mu    sync.Mutex
	cache = map[key]image.Image{}
)

// Node is one Lucide icon, size by size logical pixels, painted in ink.
// A name this set does not know leaves Src nil.
func Node(name lucide.Name, ink gui.RGB, size float32) *gui.Image {
	return &gui.Image{Src: paint(name, ink), Width: size, Height: size}
}

func paint(name lucide.Name, ink gui.RGB) image.Image {
	id := key{name: name, ink: ink}
	mu.Lock()
	defer mu.Unlock()
	if img, ok := cache[id]; ok {
		return img
	}
	img, err := lucide.Image(name, color.NRGBA{R: ink.Red, G: ink.Green, B: ink.Blue, A: ink.Alpha})
	if err != nil {
		cache[id] = nil
		return nil
	}
	cache[id] = img
	return img
}
