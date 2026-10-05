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
	"github.com/lewtec/lewkit/x/ui/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoStartsInTheCorner(t *testing.T) {
	screen := newScreen(t.Context())
	at, color := screen.logo()
	assert.Equal(t, float32(0), at.X)
	assert.Equal(t, float32(0), at.Y)
	assert.Equal(t, dvdColors[0], color)

	screen.advance(0.5)
	at, color = screen.logo()
	assert.InDelta(t, float64(pace*0.5), float64(at.X), 1e-4)
	assert.InDelta(t, float64(pace*0.5), float64(at.Y), 1e-4)
	assert.Equal(t, dvdColors[0], color)
}

func TestLogoHitsTheFarCorner(t *testing.T) {
	screen := newScreen(t.Context())
	screen.advance(5)
	at, color := screen.logo()
	_, step, ok := world.Query[glide](screen.sim.World).First(nil)
	require.True(t, ok)
	assert.Equal(t, float32(1), at.X)
	assert.Equal(t, float32(1), at.Y)
	assert.Negative(t, step.X)
	assert.Negative(t, step.Y)
	assert.Equal(t, ink{R: 255, G: 255, B: 255}, color)
}

func TestRectanglePlotsTheUsableBox(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	const width, height, bar = 160, 200, 28
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
		above := host.Front().RGBAAt(width-4, bar-2)
		edge := host.Front().RGBAAt(width-4, bar+1)
		room := host.Front().RGBAAt(width/2, bar+48)
		return above == color.RGBA{outside.Red, outside.Green, outside.Blue, outside.Alpha} &&
			edge == color.RGBA{plot.Red, plot.Green, plot.Blue, plot.Alpha} &&
			room == color.RGBA{inside.Red, inside.Green, inside.Blue, inside.Alpha}
	}, 2*time.Second, 10*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}
