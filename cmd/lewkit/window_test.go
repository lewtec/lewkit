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

func TestPaintIfResized(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_WINDOW_DRIVER", "window_mem")
	w, err := window.Open(t.Context(), window.Config{Width: 80, Height: 60})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	var last image.Point
	require.NoError(t, paintIfResized(w, &last))
	assert.Equal(t, image.Pt(80, 60), last)
	top := w.Frame().RGBAAt(40, 22)
	assert.Greater(t, int(top.R), 180, "top=%v", top)

	require.NoError(t, paintIfResized(w, &last))
	require.NoError(t, w.Resize(image.Pt(120, 90)))
	require.NoError(t, paintIfResized(w, &last))
	assert.Equal(t, image.Pt(120, 90), last)
	assert.Equal(t, 120, w.Frame().Bounds().Dx())
	assert.Equal(t, 90, w.Frame().Bounds().Dy())
	top = w.Frame().RGBAAt(60, 34)
	assert.Greater(t, int(top.R), 180, "resized top=%v", top)
}
