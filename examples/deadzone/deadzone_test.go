package main

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorWalksTheHue(t *testing.T) {
	screen := newScreen(t.Context())
	assert.Equal(t, tint{R: 255}, screen.color())

	screen.advance(0.5)
	ink := screen.color()
	assert.Equal(t, uint8(255), ink.R)
	assert.Greater(t, ink.G, uint8(0))
	assert.Equal(t, uint8(0), ink.B)
}

func TestFillCoversTheShownBox(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	const width, height, bar = 120, 160, 24
	host, err := window.Open(t.Context(), window.Config{Width: width, Height: height, Period: time.Hour})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	setter, ok := host.(interface{ SetDead([]image.Rectangle) })
	require.True(t, ok)
	setter.SetDead([]image.Rectangle{image.Rect(0, 0, width, bar)})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- gui.Run(ctx, host, ndarray.CPU, newScreen(ctx)) }()
	require.Eventually(t, func() bool {
		barPixel := host.Front().RGBAAt(8, 8)
		near := host.Front().RGBAAt(8, bar+8)
		far := host.Front().RGBAAt(width-4, height-4)
		return barPixel == color.RGBA{0, 0, 0, 255} &&
			near.R > 200 && near.A == 255 &&
			far.R > 200 && far.A == 255
	}, time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}
