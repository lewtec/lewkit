package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func TestZoomIdentity(t *testing.T) {
	face := basicfont.Face7x13
	assert.Equal(t, face, Zoom(face, 1))
	assert.Equal(t, face, Zoom(face, 1.0005))
	assert.Equal(t, face, Zoom(face, 0))
	assert.Equal(t, Use(nil), Zoom(nil, 1))
}

func TestZoomBitmap(t *testing.T) {
	face := Zoom(basicfont.Face7x13, 2)
	advance, ok := face.GlyphAdvance('M')
	require.True(t, ok)
	assert.Equal(t, 14, advance.Ceil())
	assert.Equal(t, 26, face.Metrics().Height.Ceil())
	_, mask, _, _, ok := face.Glyph(fixed.P(0, 0), 'M')
	require.True(t, ok)
	require.NotNil(t, mask)
	assert.Equal(t, 12, mask.Bounds().Dx())
	assert.Equal(t, 26, mask.Bounds().Dy())
	require.NoError(t, face.Close())
}
