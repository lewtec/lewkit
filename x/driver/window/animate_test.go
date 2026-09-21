package window_test

import (
	"context"
	"image"
	"sync/atomic"
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

func TestDriveDeliversPointer(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	got := make(chan window.Event, 1)
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- window.Drive(ctx, w, time.Hour, func(ev window.Event, _ time.Duration) error {
			if ev == nil {
				select {
				case <-ready:
				default:
					close(ready)
				}
				return nil
			}
			got <- ev
			return nil
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for drive")
	}
	type emitter interface{ Emit(window.Event) }
	w.(emitter).Emit(window.Pointer{Pos: image.Pt(1, 2), Button: 1, Pressed: true})
	select {
	case ev := <-got:
		p, ok := ev.(window.Pointer)
		require.True(t, ok)
		assert.Equal(t, image.Pt(1, 2), p.Pos)
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	cancel()
	require.NoError(t, <-done)
}

func TestDriveTicksWhileIdle(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	var ticks atomic.Int64
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- window.Drive(ctx, w, 10*time.Millisecond, func(ev window.Event, _ time.Duration) error {
			if ev == nil {
				if ticks.Add(1) == 1 {
					close(ready)
				}
			}
			return nil
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for drive")
	}
	time.Sleep(45 * time.Millisecond)
	got := ticks.Load()
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, got, int64(4), "nil-event ticks while idle")
}

func TestDriveKeepsTickingAfterEvent(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	var ticks atomic.Int64
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- window.Drive(ctx, w, 10*time.Millisecond, func(ev window.Event, _ time.Duration) error {
			if ev == nil {
				if ticks.Add(1) == 1 {
					close(ready)
				}
			}
			return nil
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for drive")
	}
	type emitter interface{ Emit(window.Event) }
	w.(emitter).Emit(window.Pointer{Pos: image.Pt(1, 1)})
	before := ticks.Load()
	time.Sleep(45 * time.Millisecond)
	got := ticks.Load()
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, got-before, int64(3), "nil-event ticks after a pointer")
}

func TestDriveTicksDuringEventFlood(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	var ticks atomic.Int64
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- window.Drive(ctx, w, 20*time.Millisecond, func(ev window.Event, _ time.Duration) error {
			if ev == nil {
				if ticks.Add(1) == 1 {
					close(ready)
				}
			}
			return nil
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for drive")
	}
	type emitter interface{ Emit(window.Event) }
	em := w.(emitter)
	deadline := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(deadline) {
		em.Emit(window.Resize{Size: image.Pt(8, 8)})
		time.Sleep(2 * time.Millisecond)
	}
	got := ticks.Load()
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, got, int64(3), "nil-event ticks during resize flood")
}

func TestAnimateStopsOnClose(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	err = window.Animate(t.Context(), w, time.Millisecond, func(*image.RGBA, time.Duration) error {
		return w.Close()
	})
	require.NoError(t, err)
}
