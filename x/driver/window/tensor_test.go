package window_test

import (
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFitCastAndResize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 3})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	src, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	pixels, err := window.Fit(src, w.Frame())
	require.NoError(t, err)
	require.Equal(t, ndarray.U8, pixels.DType())
	require.Equal(t, ndarray.Shape{3, 2, 4}, pixels.Shape())
	again, err := window.Fit(pixels, w.Frame())
	require.NoError(t, err)
	assert.True(t, again == pixels)
}

func TestPresentWritesFrame(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	src, err := ndarray.Full(200, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	pixels, err := window.Fit(src, w.Frame())
	require.NoError(t, err)
	require.NoError(t, window.Present(t.Context(), pixels, ndarray.CPU, w.Frame()))
	assert.Equal(t, color.RGBA{200, 200, 200, 200}, w.Frame().RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{200, 200, 200, 200}, w.Frame().RGBAAt(1, 1))
}
