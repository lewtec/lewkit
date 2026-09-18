package main

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
	text, err := cmd.Usage[windowCmd]("lewkit window")
	require.NoError(t, err)
	assert.Contains(t, text, "triangle")
	assert.Contains(t, text, "RGB triangle")
}

func TestPaintAndResizeEvent(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_WINDOW_DRIVER", "window_mem")
	w, err := window.Open(t.Context(), window.Config{Width: 80, Height: 60})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	evs := w.Subscribe(t.Context())
	require.NoError(t, paint(w))
	top := w.Frame().RGBAAt(40, 22)
	assert.Greater(t, int(top.R), 180, "top=%v", top)

	require.NoError(t, w.Resize(image.Pt(120, 90)))
	ev := <-evs
	_, ok := ev.(window.Resize)
	require.True(t, ok, "got %T", ev)
	require.NoError(t, handle(w, ev))
	assert.Equal(t, 120, w.Frame().Bounds().Dx())
	top = w.Frame().RGBAAt(60, 34)
	assert.Greater(t, int(top.R), 180, "resized top=%v", top)
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
