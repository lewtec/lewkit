package image

import (
	stdimage "image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColor(t *testing.T) {
	got, err := ParseColor("#0d3559")
	require.NoError(t, err)
	assert.Equal(t, Color{13, 53, 89, 255}, got)
	got, err = ParseColor("abc")
	require.NoError(t, err)
	assert.Equal(t, Color{0xaa, 0xbb, 0xcc, 255}, got)
	_, err = ParseColor("nope")
	assert.Error(t, err)
}

func TestAverage(t *testing.T) {
	img := stdimage.NewNRGBA(stdimage.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	assert.Equal(t, Color{100, 0, 0, 255}, Average(img))
}
