package png

import (
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	raw, err := Encode(src)
	require.NoError(t, err)
	require.True(t, Match(raw))
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := convert.Square(got, 1)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 10, G: 20, B: 30, A: 255}, square.NRGBAAt(0, 0))
}
