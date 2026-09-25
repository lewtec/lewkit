package image

import (
	stdimage "image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRGB(t *testing.T) {
	got, err := ParseRGB("#0d3559")
	require.NoError(t, err)
	assert.Equal(t, RGB{13, 53, 89, 255}, got)
	got, err = ParseRGB("abc")
	require.NoError(t, err)
	assert.Equal(t, RGB{0xaa, 0xbb, 0xcc, 255}, got)
	_, err = ParseRGB("nope")
	assert.Error(t, err)
}

func TestAverage(t *testing.T) {
	img := stdimage.NewNRGBA(stdimage.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	assert.Equal(t, RGB{100, 0, 0, 255}, Average(img))
}

func TestColorSpaces(t *testing.T) {
	red := RGB{255, 0, 0, 255}
	assert.Equal(t, [4]byte{0, 0, 255, 255}, red.BGR().Bytes())
	assert.Equal(t, red, red.BGR().RGB())
	assert.Equal(t, CMYK{Magenta: 255, Yellow: 255}, red.CMYK())
	assert.Equal(t, red, red.CMYK().RGB())
	assert.Equal(t, HSV{Saturation: 255, Value: 255}, red.HSV())
	assert.Equal(t, HSV{Hue: 120, Saturation: 255, Value: 255}, RGB{0, 255, 0, 255}.HSV())
	assert.Equal(t, HSV{Hue: 240, Saturation: 255, Value: 255}, RGB{0, 0, 255, 255}.HSV())
	assert.Equal(t, RGB{0, 255, 0, 255}, HSV{Hue: 120, Saturation: 255, Value: 255}.RGB())
}
