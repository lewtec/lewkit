package gui

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
)

func TestImagePaintsSource(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i] = 240
		src.Pix[i+3] = 255
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	_, err = picture.Render(&Image{Src: src, Width: 8, Height: 8}, Size{16, 16})
	require.NoError(t, err)
	ink := picture.Ink()
	require.NotNil(t, ink)
	got := ink.RGBAAt(2, 2)
	assert.Greater(t, int(got.R), 200)
	assert.Equal(t, uint8(0), got.G)
}

func TestImageRadiusClearsCorner(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i] = 255
		src.Pix[i+3] = 255
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	_, err = picture.Render(&Image{Src: src, Width: 16, Height: 16, Radius: 8}, Size{16, 16})
	require.NoError(t, err)
	ink := picture.Ink()
	require.NotNil(t, ink)
	assert.Equal(t, color.RGBA{}, ink.RGBAAt(0, 0))
	assert.NotEqual(t, color.RGBA{}, ink.RGBAAt(8, 8))
}

func TestImageClipHidesOverflow(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i] = 240
		src.Pix[i+3] = 255
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Box{Width: 32, Height: 32, Child: &Box{
		Width: 32, Height: 16, Clip: true,
		Child: &Positioned{Y: 8, Child: &Image{Src: src, Width: 16, Height: 16}},
	}}
	_, err = picture.Render(root, Size{32, 32})
	require.NoError(t, err)
	ink := picture.Ink()
	require.NotNil(t, ink)
	assert.Equal(t, color.RGBA{}, ink.RGBAAt(4, 4))
	assert.Greater(t, int(ink.RGBAAt(4, 12).R), 200)
	assert.Equal(t, color.RGBA{}, ink.RGBAAt(4, 20))
}

func TestTextColor(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Text{Value: "M", Face: basicfont.Face7x13, Ink: Color{220, 10, 10, 255}}
	_, err = picture.Render(root, Size{32, 24})
	require.NoError(t, err)
	ink := picture.Ink()
	require.NotNil(t, ink)
	var red int
	for y := 0; y < ink.Bounds().Dy(); y++ {
		for x := 0; x < ink.Bounds().Dx(); x++ {
			c := ink.RGBAAt(x, y)
			if c.R > 150 && c.G < 40 {
				red++
			}
		}
	}
	assert.Greater(t, red, 5)
}
