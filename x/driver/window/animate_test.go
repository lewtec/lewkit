package window_test

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnimateStopsOnCancel(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	n := 0
	err = window.Animate(ctx, w, time.Millisecond, func(*image.RGBA, time.Duration) error {
		n++
		if n >= 3 {
			cancel()
		}
		return nil
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 3)
}

func TestAnimatePaintsOnResize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sizes := make(chan image.Point, 8)
	done := make(chan error, 1)
	go func() {
		done <- window.Animate(ctx, w, time.Hour, func(dst *image.RGBA, _ time.Duration) error {
			sizes <- dst.Bounds().Size()
			return nil
		})
	}()
	select {
	case got := <-sizes:
		require.Equal(t, image.Pt(8, 8), got)
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first paint")
	}
	require.NoError(t, w.Resize(image.Pt(16, 10)))
	select {
	case got := <-sizes:
		require.Equal(t, image.Pt(16, 10), got)
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for resize paint")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestAnimateStopsOnClose(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	err = window.Animate(t.Context(), w, time.Millisecond, func(*image.RGBA, time.Duration) error {
		return w.Close()
	})
	require.NoError(t, err)
}
