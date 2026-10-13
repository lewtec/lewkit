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

func TestLogoLeavesTheCorner(t *testing.T) {
	screen := newScreen(t.Context())
	at, color := screen.logo()
	assert.Equal(t, float32(0), at.X)
	assert.Equal(t, float32(0), at.Y)
	assert.Equal(t, dvdColors[0], color)

	screen.advance(0.5)
	at, color = screen.logo()
	assert.InDelta(t, float64(paceX*0.5), float64(at.X), 1e-4)
	assert.InDelta(t, float64(paceY*0.5), float64(at.Y), 1e-4)
	assert.Greater(t, at.X, at.Y)
	assert.Equal(t, dvdColors[0], color)
}

func TestLogoChangesCourseAtAWall(t *testing.T) {
	screen := newScreen(t.Context())
	screen.advance(3)
	at, color := screen.logo()
	_, step, ok := world.Query[glide](screen.sim.World).First(nil)
	require.True(t, ok)
	assert.InDelta(t, float64(2-paceX*3), float64(at.X), 1e-4)
	assert.InDelta(t, float64(paceY*3), float64(at.Y), 1e-4)
	assert.InDelta(t, float64(-kick[1]), float64(step.X), 1e-5)
	assert.InDelta(t, float64(paceY), float64(step.Y), 1e-5)
	want := dvdColors[1]
	want.N = 1
	assert.Equal(t, want, color)

	// The far horizontal edge arrives on its own. The vertical pace stays.
	screen.advance(2.6)
	at, color = screen.logo()
	_, step, ok = world.Query[glide](screen.sim.World).First(nil)
	require.True(t, ok)
	assert.Greater(t, at.X, float32(0))
	assert.Less(t, at.Y, float32(1))
	assert.InDelta(t, float64(-kick[1]), float64(step.X), 1e-5)
	assert.InDelta(t, float64(-kick[5]), float64(step.Y), 1e-5)
	want = dvdColors[2]
	want.N = 2
	assert.Equal(t, want, color)
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
