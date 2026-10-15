package lucide

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert/svg/render"
	"github.com/stretchr/testify/require"
)

func TestIconInlinesCurrentColor(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Icon(NameSearch).Render(t.Context(), &buf))
	text := buf.String()
	require.Contains(t, text, `stroke="currentColor"`)
	require.Contains(t, text, "<circle ")
}

func TestIconUnknown(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Icon(Name("not-an-icon")).Render(t.Context(), &buf))
	require.Empty(t, buf.String())
}

func TestImageSearchIsRed(t *testing.T) {
	t.Parallel()
	img, err := Image(NameSearch, color.NRGBA{R: 255, A: 255})
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 24, 24), img.Bounds())
	require.True(t, redInk(img), "search icon has no red pixel")
}

func TestImageUnknown(t *testing.T) {
	t.Parallel()
	_, err := Image(Name("not-an-icon"), color.Black)
	require.ErrorIs(t, err, ErrName)
}

func TestParseName(t *testing.T) {
	t.Parallel()
	got, err := ParseName("search")
	require.NoError(t, err)
	require.Equal(t, NameSearch, got)
	require.Equal(t, "search", got.String())
	_, err = ParseName("not-an-icon")
	require.ErrorIs(t, err, ErrName)
}

func TestImageNilColor(t *testing.T) {
	t.Parallel()
	img, err := Image(NameSearch, nil)
	require.NoError(t, err)
	require.True(t, ink(img))
}

// TestDecodeEveryIcon is the fixture for render.Decode.
// Each icon is prepared first: the root stroke is copied onto the shapes,
// and line and polyline become paths.
func TestDecodeEveryIcon(t *testing.T) {
	t.Parallel()
	names := Name("").Values()
	require.Equal(t, iconCount, len(iconBody))
	require.Len(t, names, iconCount)
	for _, name := range names {
		t.Run(name.String(), func(t *testing.T) {
			body, ok := SVG(name)
			require.True(t, ok)
			painted, err := guiSVG(body, color.NRGBA{A: 255})
			require.NoError(t, err)
			img, err := render.Decode([]byte(painted))
			require.NoError(t, err)
			require.Equal(t, image.Rect(0, 0, 24, 24), img.Bounds())
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
