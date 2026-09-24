package gui

import "github.com/lewtec/lewkit/x/driver/colorscheme"

// SchemeMsg is a system color-scheme change. Run sends the current scheme
// before the first frame, then again when it changes.
type SchemeMsg struct {
	Scheme colorscheme.Scheme
}

// Palette is the window fill and the text ink for scheme.
// An unset scheme keeps the dark pair the built-in widgets already used.
func Palette(scheme colorscheme.Scheme) (background, ink Color) {
	if scheme == colorscheme.Light {
		return Color{246, 246, 244, 255}, Color{28, 28, 34, 255}
	}
	return Color{28, 28, 34, 255}, Color{238, 238, 238, 255}
}
