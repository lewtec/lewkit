package window_test

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenFrameDrawResize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Title: "t", Width: 8, Height: 6})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	frame := w.Frame()
	require.Equal(t, 8, frame.Bounds().Dx())
	require.Equal(t, 6, frame.Bounds().Dy())

	red := image.NewUniform(color.RGBA{R: 255, A: 255})
	draw.Draw(frame, frame.Bounds(), red, image.Point{}, draw.Src)
	require.NoError(t, w.Draw())
	assert.Equal(t, color.RGBA{R: 255, A: 255}, frame.RGBAAt(0, 0))

	require.NoError(t, w.Resize(image.Pt(12, 4)))
	frame = w.Frame()
	assert.Equal(t, 12, frame.Bounds().Dx())
	assert.Equal(t, 4, frame.Bounds().Dy())
	assert.Equal(t, color.RGBA{R: 255, A: 255}, frame.RGBAAt(0, 0))
}

func TestOpenDefaultSize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	frame := w.Frame()
	assert.Equal(t, 640, frame.Bounds().Dx())
	assert.Equal(t, 480, frame.Bounds().Dy())
}

func TestOpenRejectsNegativeSize(t *testing.T) {
	_, err := window.Open(t.Context(), window.Config{Width: -1, Height: 10})
	assert.ErrorIs(t, err, window.ErrSize)
}

func TestDrawAfterClose(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.ErrorIs(t, w.Draw(), window.ErrClosed)
	assert.ErrorIs(t, w.Resize(image.Pt(3, 3)), window.ErrClosed)
}

func TestResizeRejectsNonPositive(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.ErrorIs(t, w.Resize(image.Pt(0, 2)), window.ErrSize)
	assert.ErrorIs(t, w.Resize(image.Pt(2, -1)), window.ErrSize)
}

func TestSubscribeCancelCloses(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 4})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	ch := w.Subscribe(ctx)
	cancel()
	_, ok := <-ch
	assert.False(t, ok)
}

func TestToBGRA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})
	dst := make([]byte, 4)
	window.ToBGRA(dst, src)
	assert.Equal(t, []byte{3, 2, 1, 4}, dst)
}
