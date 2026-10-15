package ico

import (
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	raw, err := Encode(solid(color.NRGBA{R: 255, G: 0, B: 0, A: 255}), []int{16})
	require.NoError(t, err)
	require.True(t, Match(raw))
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := convert.Square(got, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 255, G: 0, B: 0, A: 255}, square.NRGBAAt(8, 8))
}

func TestDIB32(t *testing.T) {
	raw := make([]byte, 40+8)
	raw[0] = 40
	raw[4] = 1
	raw[8] = 2
	raw[12] = 1
	raw[14] = 32
	raw[40] = 10
	raw[41] = 20
	raw[42] = 30
	raw[43] = 255
	got, err := decodeDIB(raw)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 30, G: 20, B: 10, A: 255}, got.(*image.NRGBA).NRGBAAt(0, 0))
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
