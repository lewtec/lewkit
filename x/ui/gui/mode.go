package gui

import "github.com/lewtec/lewkit/x/driver/daynight"

// ModeMsg is a day or night change. Run sends the current mode
// before the first frame, then again when it changes.
type ModeMsg struct {
	Mode daynight.Mode
}

// Palette is the window fill and the text ink for mode.
// An unset mode keeps the dark pair the built-in widgets already used.
func Palette(mode daynight.Mode) (background, ink Color) {
	if mode == daynight.Light {
		return Color{246, 246, 244, 255}, Color{28, 28, 34, 255}
	}
	return Color{28, 28, 34, 255}, Color{238, 238, 238, 255}
}
