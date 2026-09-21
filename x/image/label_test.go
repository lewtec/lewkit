package image

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
)

func TestLabel(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 160, 48))
	Label(dst, 8, 24, "60 fps")
	var lit int
	for y := 0; y < 48; y++ {
		for x := 0; x < 120; x++ {
			c := dst.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) > 0 {
				lit++
			}
		}
	}
	assert.Greater(t, lit, 20)
}

func TestFaceLoads(t *testing.T) {
	f := Face()
	require.NotNil(t, f)
	assert.Greater(t, LineHeight(nil), 8)
	assert.Greater(t, Advance('M', nil), 2)
	assert.Equal(t, 7, Advance('M', basicfont.Face7x13))
}
