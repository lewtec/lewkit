package lucide

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIconInlinesCurrentColor(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Icon("search").Render(t.Context(), &buf))
	text := buf.String()
	require.Contains(t, text, `stroke="currentColor"`)
	require.Contains(t, text, "<circle ")
}

func TestIconUnknown(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Icon("not-an-icon").Render(t.Context(), &buf))
	require.Empty(t, buf.String())
}

func TestImageSearchIsRed(t *testing.T) {
	t.Parallel()
	img, err := Image("search", color.NRGBA{R: 255, A: 255})
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 24, 24), img.Bounds())
	require.True(t, redInk(img), "search icon has no red pixel")
}

func TestImageUnknown(t *testing.T) {
	t.Parallel()
	_, err := Image("not-an-icon", color.Black)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestImageNilColor(t *testing.T) {
	t.Parallel()
	img, err := Image("search", nil)
	require.NoError(t, err)
	require.True(t, ink(img))
}

func TestImageEveryIcon(t *testing.T) {
	require.Equal(t, iconCount, len(iconBody))
	require.Len(t, Names(), iconCount)
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			img, err := Image(name, color.Black)
			require.NoError(t, err)
			require.Truef(t, ink(img), "%s painted nothing", name)
		})
	}
}

func redInk(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a > 0 && r > g && r > bl {
				return true
			}
		}
	}
	return false
}

func ink(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				return true
			}
		}
	}
	return false
}
