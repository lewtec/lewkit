package image

import (
	"image"
	"image/color"
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

func TestStampColor(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 80, 24))
	Stamp{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{R: 200, G: 0, B: 0, A: 255}),
		X:    1,
		Y:    12,
		Text: "M",
		Face: basicfont.Face7x13,
	}.Draw()
	var red int
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			c := dst.RGBAAt(x, y)
			if c.R > 150 && c.G < 40 {
				red++
			}
		}
	}
	assert.Greater(t, red, 5)
}

func TestFaceLoads(t *testing.T) {
	f := Face()
	require.NotNil(t, f)
	assert.Greater(t, LineHeight(nil), 8)
	assert.Greater(t, Advance('M', nil), 2)
	assert.Equal(t, 7, Advance('M', basicfont.Face7x13))
}

func TestFaceSizeGrows(t *testing.T) {
	small := FaceSize(16)
	large := FaceSize(40)
	require.NotNil(t, small)
	require.NotNil(t, large)
	if small == basicfont.Face7x13 {
		assert.Equal(t, large, small)
		return
	}
	assert.Greater(t, LineHeight(large), LineHeight(small))
	assert.Greater(t, Advance('M', large), Advance('M', small))
}
