package icns

import (
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	raw, err := Encode(solid(color.NRGBA{R: 0, G: 0, B: 255, A: 255}), []Spec{{"icp4", 16}})
	require.NoError(t, err)
	require.Equal(t, "icns", string(raw[:4]))
	require.True(t, Match(raw))
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := convert.Square(got, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 0, G: 0, B: 255, A: 255}, square.NRGBAAt(0, 0))
}

func solid(c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}
