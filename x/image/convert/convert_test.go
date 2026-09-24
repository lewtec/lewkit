package convert

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePNG(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	raw, err := EncodePNG(src)
	require.NoError(t, err)
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := Square(got, 1)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 10, G: 20, B: 30, A: 255}, square.NRGBAAt(0, 0))
}

func TestDecodeJPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}))
	got, err := Decode(buf.Bytes())
	require.NoError(t, err)
	square, err := Square(got, 8)
	require.NoError(t, err)
	require.Equal(t, 8, square.Bounds().Dx())
	require.Equal(t, 8, square.Bounds().Dy())
}

func TestICORoundTrip(t *testing.T) {
	raw, err := EncodeICO(solid(color.NRGBA{R: 255, G: 0, B: 0, A: 255}), []int{16})
	require.NoError(t, err)
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := Square(got, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 255, G: 0, B: 0, A: 255}, square.NRGBAAt(8, 8))
}

func TestICNSRoundTrip(t *testing.T) {
	raw, err := EncodeICNS(solid(color.NRGBA{R: 0, G: 0, B: 255, A: 255}), []ICNSSpec{{"icp4", 16}})
	require.NoError(t, err)
	require.Equal(t, "icns", string(raw[:4]))
	got, err := Decode(raw)
	require.NoError(t, err)
	square, err := Square(got, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 0, G: 0, B: 255, A: 255}, square.NRGBAAt(0, 0))
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

func TestRejectUnknownBytes(t *testing.T) {
	_, err := Decode([]byte("not an image"))
	require.ErrorIs(t, err, ErrFormat)
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
