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

func TestAnimateStopsOnClose(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	err = window.Animate(t.Context(), w, time.Millisecond, func(*image.RGBA, time.Duration) error {
		return w.Close()
	})
	require.NoError(t, err)
}
