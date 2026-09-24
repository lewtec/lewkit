package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
)

func TestTextPartialClip(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	text := &Text{Value: "M", Face: basicfont.Face7x13, Ink: RGB{220, 10, 10, 255}}
	root := &Box{Width: 40, Height: 24, Child: &Box{
		Width: 40, Height: 8, Clip: true,
		Child: &Positioned{Y: -6, Child: text},
	}}
	_, err = picture.Render(root, Size{40, 24})
	require.NoError(t, err)
	ink := picture.Ink()
	require.NotNil(t, ink)
	var shown int
	for y := 0; y < 8; y++ {
		for x := 0; x < 20; x++ {
			if ink.RGBAAt(x, y).R > 150 {
				shown++
			}
		}
	}
	assert.Greater(t, shown, 0)
	for y := 8; y < 24; y++ {
		for x := 0; x < 40; x++ {
			assert.Equal(t, uint8(0), ink.RGBAAt(x, y).R)
		}
	}
}
