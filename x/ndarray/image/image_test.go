package image

import (
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	expr, err := Fill(2, 2, 10, 20, 30, 255)
	require.NoError(t, err)
	dst, err := Raster(t.Context(), expr, ndarray.CPU)
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{10, 20, 30, 255}, dst.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{10, 20, 30, 255}, dst.RGBAAt(1, 1))
}

func TestWrite(t *testing.T) {
	pixels := []uint8{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 0, 0, 0, 255}
	dst := RGBA(2, 2, pixels)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, dst.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, dst.RGBAAt(1, 0))
}
