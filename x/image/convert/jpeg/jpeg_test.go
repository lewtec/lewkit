package jpeg

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	raw, err := Encode(src)
	require.NoError(t, err)
	require.True(t, Match(raw))
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := convert.Square(got, 8)
	require.NoError(t, err)
	require.Equal(t, 8, square.Bounds().Dx())
	require.Equal(t, 8, square.Bounds().Dy())
}
