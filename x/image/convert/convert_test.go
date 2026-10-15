package convert

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPadCentersWideImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	sq := Pad(img)
	require.Equal(t, 100, sq.Bounds().Dx())
	require.Equal(t, 100, sq.Bounds().Dy())
	require.Equal(t, color.NRGBA{}, sq.NRGBAAt(0, 0))
	require.Equal(t, color.NRGBA{R: 255, A: 255}, sq.NRGBAAt(0, 25))
}

func TestARGBStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(5, 7, 6, 8))
	src.SetNRGBA(5, 7, color.NRGBA{R: 255, G: 0, B: 0, A: 128})
	require.Equal(t, []byte{128, 255, 0, 0}, ARGB(src))
}

func TestSquareKeepsAspect(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	got, err := Square(src, 4)
	require.NoError(t, err)
	require.Equal(t, uint8(0), got.NRGBAAt(0, 0).A)
}
