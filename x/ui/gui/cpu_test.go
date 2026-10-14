package gui

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
)

func TestCompositeMatchesKernel(t *testing.T) {
	roots := []Node{
		&Box{Fill: &RGB{255, 0, 0, 128}},
		&Box{Fill: &RGB{0, 255, 0, 255}, Radius: 20},
		&Stack{Children: []Node{
			&Box{Fill: &RGB{255, 0, 0, 255}},
			&Positioned{X: 0, Y: 0, Child: &Box{Width: 8, Height: 8, Fill: &RGB{0, 0, 255, 128}}},
		}},
		&Text{Value: "M", Face: basicfont.Face7x13, Ink: RGB{220, 10, 10, 255}},
		&Box{
			Fill: &RGB{28, 24, 32, 255},
			Child: &Stack{Children: []Node{
				&Box{Fill: &RGB{12, 12, 16, 255}},
				&Positioned{X: 4, Y: 6, Child: &Box{Width: 18, Height: 10, Fill: &RGB{240, 80, 40, 255}, Child: &Text{Value: "DVD", Face: basicfont.Face7x13, Ink: RGB{255, 255, 255, 255}}}},
			}},
		},
	}
	for _, root := range roots {
		want := raster(t, root, 40, 32)
		got := compositeOf(t, root, 40, 32)
		require.Equal(t, len(want), len(got))
		for i := range want {
			delta := int(want[i]) - int(got[i])
			if delta < 0 {
				delta = -delta
			}
			require.LessOrEqual(t, delta, 1, "byte %d", i)
		}
	}
}

func compositeOf(t *testing.T, root Node, width, height int) []uint8 {
	t.Helper()
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	picture.holdList = true
	_, err = picture.Render(root, Size{float32(width), float32(height)})
	require.NoError(t, err)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	var ink *image.RGBA
	if picture.hadInk {
		ink = picture.inkRGBA
	}
	composite(dst, picture.fills, nil, ink)
	return append([]uint8(nil), dst.Pix...)
}
