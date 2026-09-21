package window_test

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFramePeriodFromConfig(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8, Period: time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.Equal(t, time.Millisecond, w.FramePeriod())
}

func TestFramePeriodDefault(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.Equal(t, window.DefaultFramePeriod, w.FramePeriod())
}

func TestOpenFrameDrawResize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Title: "t", Width: 8, Height: 6})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	require.Equal(t, image.Pt(8, 6), w.Size())
	frame := w.Frame()
	require.Equal(t, 8, frame.Bounds().Dx())
	require.Equal(t, 6, frame.Bounds().Dy())

	red := image.NewUniform(color.RGBA{R: 255, A: 255})
	draw.Draw(frame, frame.Bounds(), red, image.Point{}, draw.Src)
	require.NoError(t, w.Draw())
	assert.Equal(t, color.RGBA{R: 255, A: 255}, frame.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{R: 255, A: 255}, w.Front().RGBAAt(0, 0))

	require.NoError(t, w.Resize(image.Pt(12, 4)))
	frame = w.Frame()
	assert.Equal(t, 12, frame.Bounds().Dx())
	assert.Equal(t, 4, frame.Bounds().Dy())
	assert.Equal(t, color.RGBA{R: 255, A: 255}, w.Front().RGBAAt(0, 0))
}

func TestDrawSwapsPages(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	a := w.Frame()
	a.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	require.NoError(t, w.Draw())
	b := w.Frame()
	assert.False(t, a == b)
	front := w.Front()
	assert.True(t, a == front)
	assert.Equal(t, color.RGBA{R: 255, A: 255}, front.RGBAAt(0, 0))
}

func TestOpenDefaultSize(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	frame := w.Frame()
	assert.Equal(t, 640, frame.Bounds().Dx())
	assert.Equal(t, 480, frame.Bounds().Dy())
}

func TestOpenRejectsNegativeSize(t *testing.T) {
	_, err := window.Open(t.Context(), window.Config{Width: -1, Height: 10})
	assert.ErrorIs(t, err, window.ErrSize)
}

func TestSwapBlit(t *testing.T) {
	buf := window.NewBuffer(2, 2)
	called := 0
	require.NoError(t, window.SwapBlit(buf, func() error {
		called++
		return nil
	}))
	assert.Equal(t, 1, called)
	require.NoError(t, buf.Close())
	assert.ErrorIs(t, window.SwapBlit(buf, func() error {
		t.Fatal("blit on closed buffer")
		return nil
	}), window.ErrClosed)
}

func TestWantSizeSet(t *testing.T) {
	var want window.WantSize
	assert.False(t, want.Set(0, 10))
	assert.Equal(t, image.Pt(1, 1), want.Point(image.Pt(1, 1)))
	assert.True(t, want.Set(8, 6))
	assert.Equal(t, image.Pt(8, 6), want.Point(image.Pt(1, 1)))
	assert.False(t, want.Set(8, 6))
	assert.True(t, want.Set(10, 6))
	assert.Equal(t, image.Pt(10, 6), want.Point(image.Pt(1, 1)))
}

func TestHostSizeAndFrame(t *testing.T) {
	buf := window.NewBuffer(4, 3)
	var mu sync.Mutex
	var want window.WantSize
	assert.Equal(t, image.Pt(4, 3), window.HostSize(buf, &mu, &want))
	assert.True(t, want.Set(8, 6))
	assert.Equal(t, image.Pt(8, 6), window.HostSize(buf, &mu, &want))

	frame := window.HostFrame(buf, image.Point{})
	require.Equal(t, image.Pt(4, 3), frame.Bounds().Size())
	frame = window.HostFrame(buf, image.Pt(8, 6))
	require.Equal(t, image.Pt(8, 6), frame.Bounds().Size())
	assert.Equal(t, image.Pt(8, 6), buf.Size())

	require.NoError(t, buf.Close())
	frame = window.HostFrame(buf, image.Pt(10, 10))
	require.NotNil(t, frame)
	assert.Equal(t, image.Pt(8, 6), frame.Bounds().Size())
}

func TestSetWantEmitsResize(t *testing.T) {
	buf := window.NewBuffer(8, 6)
	var mu sync.Mutex
	var want window.WantSize
	ch := buf.Subscribe(t.Context())
	window.SetWant(buf, &mu, &want, 10, 12)
	ev := <-ch
	assert.Equal(t, window.Resize{Size: image.Pt(10, 12)}, ev)
	window.SetWant(buf, &mu, &want, 10, 12)
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event %v", ev)
	default:
	}
}

func TestCloseWhenDone(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	window.CloseWhenDone(ctx, w)
	cancel()
	require.Eventually(t, func() bool {
		return w.Draw() == window.ErrClosed
	}, time.Second, 5*time.Millisecond)
}

func TestCloseWhenDoneNilCtx(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	window.CloseWhenDone(nil, w)
	require.NoError(t, w.Draw())
}

func TestDrawAfterClose(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.ErrorIs(t, w.Draw(), window.ErrClosed)
	assert.ErrorIs(t, w.Resize(image.Pt(3, 3)), window.ErrClosed)
}

func TestResizeRejectsNonPositive(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.ErrorIs(t, w.Resize(image.Pt(0, 2)), window.ErrSize)
	assert.ErrorIs(t, w.Resize(image.Pt(2, -1)), window.ErrSize)
}

func TestSubscribePointer(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	ch := w.Subscribe(t.Context())
	w.(interface{ Emit(window.Event) }).Emit(window.Pointer{Pos: image.Pt(3, 4), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	ev := <-ch
	p, ok := ev.(window.Pointer)
	require.True(t, ok)
	assert.Equal(t, image.Pt(3, 4), p.Pos)
	assert.Equal(t, 1, p.Button)
	assert.True(t, p.Pressed)
}

func TestSubscribeCancelCloses(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 4})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	ch := w.Subscribe(ctx)
	cancel()
	_, ok := <-ch
	assert.False(t, ok)
}

func TestToBGRA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})
	dst := make([]byte, 4)
	window.ToBGRA(dst, src)
	assert.Equal(t, []byte{3, 2, 1, 4}, dst)
}
