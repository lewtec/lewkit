package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRasterPNGBytes(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	raw, err := EncodePNG(src)
	require.NoError(t, err)
	got, err := Raster(Icon{Bytes: raw}, 1)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 10, G: 20, B: 30, A: 255}, got.NRGBAAt(0, 0))
}

func TestRasterJPEGBytes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}))
	got, err := Raster(Icon{Bytes: buf.Bytes()}, 8)
	require.NoError(t, err)
	require.Equal(t, 8, got.Bounds().Dx())
	require.Equal(t, 8, got.Bounds().Dy())
}

func TestICORoundTrip(t *testing.T) {
	src := solid(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	raw, err := EncodeICO(src, []int{16})
	require.NoError(t, err)
	got, err := Raster(Icon{Bytes: raw}, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 255, G: 0, B: 0, A: 255}, got.NRGBAAt(8, 8))
}

func TestICNSRoundTrip(t *testing.T) {
	src := solid(color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	raw, err := EncodeICNS(src, []struct {
		Type string
		Size int
	}{{"icp4", 16}})
	require.NoError(t, err)
	require.Equal(t, "icns", string(raw[:4]))
	got, err := Raster(Icon{Bytes: raw}, 16)
	require.NoError(t, err)
	require.Equal(t, color.NRGBA{R: 0, G: 0, B: 255, A: 255}, got.NRGBAAt(0, 0))
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

func TestARGBStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(5, 7, 6, 8))
	src.SetNRGBA(5, 7, color.NRGBA{R: 255, G: 0, B: 0, A: 128})
	got := ARGB(src)
	require.Equal(t, []byte{128, 255, 0, 0}, got)
}

func TestPadKeepsAspect(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	got, err := Raster(Icon{Image: src}, 4)
	require.NoError(t, err)
	require.Equal(t, 4, got.Bounds().Dx())
	require.Equal(t, uint8(0), got.NRGBAAt(0, 0).A)
}

func TestNameOnlyIcon(t *testing.T) {
	got, err := Raster(Icon{Name: "applications-system"}, 32)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRejectUnknownBytes(t *testing.T) {
	_, err := Raster(Icon{Bytes: []byte("not an image")}, 16)
	require.ErrorIs(t, err, ErrIcon)
}

func TestMenuTreeIDs(t *testing.T) {
	nodes := Tree([]Item{
		{Label: "Open"},
		{Label: "More", Children: []Item{{Label: "About"}, {Separator: true}}},
		{Label: "Quit"},
	})
	require.Equal(t, 1, nodes[0].ID)
	require.Equal(t, 2, nodes[1].ID)
	require.Equal(t, 3, nodes[1].Children[0].ID)
	require.Equal(t, 4, nodes[1].Children[1].ID)
	require.Equal(t, 5, nodes[2].ID)
	item, ok := Find(nodes, 3)
	require.True(t, ok)
	require.Equal(t, "About", item.Label)
	_, ok = Find(nodes, 9)
	require.False(t, ok)
}

func TestDIB32(t *testing.T) {
	// 1×1 BGRA bottom-up, height doubled the way an ICO mask does.
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
