package image

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabel(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 80, 40))
	Label(dst, 8, 16, "60 fps")
	var lit int
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			c := dst.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) > 0 {
				lit++
			}
		}
	}
	assert.Greater(t, lit, 20)
}
