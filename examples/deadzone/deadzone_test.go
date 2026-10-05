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

func TestSquareDriftsAndBounces(t *testing.T) {
	screen := newScreen(t.Context())
	at, ink := screen.square()
	assert.Equal(t, float32(0), at.X)
	assert.Equal(t, float32(0), at.Y)
	assert.Equal(t, tint{R: 255}, ink)

	screen.advance(0.5)
	at, ink = screen.square()
	assert.InDelta(t, float64(paceX*0.5), float64(at.X), 1e-4)
	assert.InDelta(t, float64(paceY*0.5), float64(at.Y), 1e-4)
	assert.NotEqual(t, tint{R: 255}, ink)
}

func TestSquareBouncesOffTheFarEdge(t *testing.T) {
	screen := newScreen(t.Context())
	// 4s carries X past the far edge and leaves Y short of it.
	screen.advance(4)
	at, _ := screen.square()
	_, step, ok := world.Query[glide](screen.sim.World).First(nil)
	require.True(t, ok)
	assert.Equal(t, float32(1), at.X)
	assert.Negative(t, step.X)
	assert.InDelta(t, 4*float64(paceY), float64(at.Y), 1e-3)
	assert.Positive(t, step.Y)
}

func TestSquareStaysOutOfTheBar(t *testing.T) {
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
	model := newScreen(ctx)
	done := make(chan error, 1)
	go func() { done <- gui.Run(ctx, host, ndarray.CPU, model) }()
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(8, 8) == color.RGBA{0, 0, 0, 255} &&
			host.Front().RGBAAt(8, bar+8).R > 200 &&
			host.Front().RGBAAt(8, bar+8).A == 255
	}, time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}
