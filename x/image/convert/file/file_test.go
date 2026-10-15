package file

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/image/convert"
	pngcodec "github.com/lewtec/lewkit/x/image/convert/png"
	"github.com/stretchr/testify/require"
)

func TestDecodePNG(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	raw, err := pngcodec.Encode(src)
	require.NoError(t, err)
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := convert.Square(got, 1)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 10, G: 20, B: 30, A: 255}, square.NRGBAAt(0, 0))
}

func TestOpenPNG(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	path := filepath.Join(t.TempDir(), "mark.png")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, src))
	require.NoError(t, f.Close())
	got, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, 2, got.Bounds().Dx())
}

func TestRejectUnknownBytes(t *testing.T) {
	_, err := Decode([]byte("not an image"))
	require.ErrorIs(t, err, convert.ErrFormat)
}
