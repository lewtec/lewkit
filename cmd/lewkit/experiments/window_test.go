package experiments

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowUsage(t *testing.T) {
	text, err := cmd.Usage[Window]("lewkit experiments window")
	require.NoError(t, err)
	assert.Contains(t, text, "triangle")
	assert.Contains(t, text, "one turn")
}

func TestPaintAndResizeEvent(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_WINDOW_DRIVER", "window_mem")
	w, err := window.Open(t.Context(), window.Config{Width: 80, Height: 60})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	evs := w.Subscribe(t.Context())
	require.NoError(t, paint(w, 0, 0))
	top := lastFrame(w).RGBAAt(40, 16)
	assert.Greater(t, int(top.R), 180, "top=%v", top)

	require.NoError(t, w.Resize(image.Pt(120, 90)))
	ev := <-evs
	_, ok := ev.(window.Resize)
	require.True(t, ok, "got %T", ev)
	require.NoError(t, paint(w, 0, 0))
	assert.Equal(t, 120, w.Frame().Bounds().Dx())
	top = lastFrame(w).RGBAAt(60, 22)
	assert.Greater(t, int(top.R), 180, "resized top=%v", top)
}

func TestPaintRotates(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_WINDOW_DRIVER", "window_mem")
	w, err := window.Open(t.Context(), window.Config{Width: 80, Height: 60})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	require.NoError(t, paint(w, 0, 0))
	a := lastFrame(w).RGBAAt(40, 16)
	require.NoError(t, paint(w, 0.5, 0))
	b := lastFrame(w).RGBAAt(40, 16)
	assert.NotEqual(t, a, b)
}

func TestDrawFPS(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 80, 40))
	drawFPS(dst, 60)
	var lit int
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			c := dst.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) > 0 {
				lit++
			}
		}
	}
	assert.Greater(t, lit, 20)
}

func TestSubscribeClose(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_WINDOW_DRIVER", "window_mem")
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	evs := w.Subscribe(t.Context())
	require.NoError(t, w.Close())
	ev := <-evs
	_, ok := ev.(window.Close)
	assert.True(t, ok, "got %T", ev)
}
