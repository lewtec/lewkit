package scene

import "github.com/lewtec/lewkit/x/ui/gui"

// TriangleModel is the RGB triangle, one turn every four seconds.
func TriangleModel(width, height int) (gui.Model, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return newSpinModel(triangleTurnsPerSecond, width, height, triangleDynamic)
}

// PerlinModel animates Perlin noise.
func PerlinModel(width, height int) (gui.Model, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return newFrameModel(0.4, width, height, perlinDynamic)
}

// FractalModel animates a Julia set, one revolution every twenty seconds.
func FractalModel(width, height int) (gui.Model, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return newFrameModel(fractalTurnsPerSecond, width, height, fractalDynamic)
}
