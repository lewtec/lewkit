package image

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriangleColors(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	Triangle(dst)

	top := dst.RGBAAt(100, 70)
	assert.Greater(t, int(top.R), 180, "top=%v", top)
	assert.Less(t, int(top.G)+int(top.B), 80, "top=%v", top)

	br := dst.RGBAAt(130, 140)
	assert.Greater(t, int(br.G), 180, "br=%v", br)
	assert.Less(t, int(br.R)+int(br.B), 80, "br=%v", br)

	bl := dst.RGBAAt(70, 140)
	assert.Greater(t, int(bl.B), 180, "bl=%v", bl)
	assert.Less(t, int(bl.R)+int(bl.G), 80, "bl=%v", bl)

	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(199, 0))
}

func TestTriangleOffsetBounds(t *testing.T) {
	dst := image.NewRGBA(image.Rect(10, 20, 110, 120))
	Triangle(dst)
	require.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(10, 20))
	top := dst.RGBAAt(60, 55)
	assert.Greater(t, int(top.R), 180, "offset top=%v", top)
}

func TestTriangleEmpty(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 0, 0))
	Triangle(dst)
}

func TestTriangleTurnHalf(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	TriangleTurn(dst, 0.5)
	bot := dst.RGBAAt(100, 130)
	assert.Greater(t, int(bot.R), 180, "half turn bottom=%v", bot)
	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(100, 20))
}
