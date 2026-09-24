package gui

import (
	"context"
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunNilModel(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 4})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	assert.ErrorIs(t, Run(t.Context(), host, ndarray.CPU, nil), ErrModel)
}

func TestRunNilWindow(t *testing.T) {
	solid, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	assert.ErrorIs(t, Run(t.Context(), nil, ndarray.CPU, solid), window.ErrClosed)
}

func TestPictureFrameSig(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Box{Fill: &Color{10, 20, 30, 255}}
	_, err = picture.Render(root, Size{4, 4})
	require.NoError(t, err)
	first := picture.frameSig()
	_, err = picture.Render(root, Size{4, 4})
	require.NoError(t, err)
	assert.Equal(t, first, picture.frameSig())
	root.Fill = &Color{1, 2, 3, 255}
	_, err = picture.Render(root, Size{4, 4})
	require.NoError(t, err)
	assert.NotEqual(t, first, picture.frameSig())
}

func TestInkStaysWhenDrawsMatch(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	root := &Text{Value: "Hi", Ink: Color{255, 255, 255, 255}}
	_, err = picture.Render(root, Size{64, 32})
	require.NoError(t, err)
	assert.True(t, picture.inkFresh)
	picture.recordOnly = true
	_, err = picture.Render(root, Size{64, 32})
	require.NoError(t, err)
	assert.False(t, picture.inkFresh)
}

func TestPictureFrameSigUsesDraws(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	root := &Text{Value: "Clocks", Ink: Color{255, 255, 255, 255}}
	_, err = picture.Render(root, Size{1800, 1200})
	require.NoError(t, err)
	first := picture.frameSig()
	picture.recordOnly = true
	_, err = picture.Render(root, Size{1800, 1200})
	require.NoError(t, err)
	assert.Equal(t, first, picture.frameSig())
	root.Value = "Clocks!"
	picture.recordOnly = true
	_, err = picture.Render(root, Size{1800, 1200})
	require.NoError(t, err)
	assert.NotEqual(t, first, picture.frameSig())
}

func TestSolidView(t *testing.T) {
	solid, err := NewSolid(10, 20, 30, 255)
	require.NoError(t, err)
	require.NotNil(t, solid.View())
}

func TestRunPaintsSolid(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	solid, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, host, ndarray.CPU, solid) }()
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255} &&
			host.Front().RGBAAt(3, 2) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

type fpsLog struct {
	inner Model
	mu    sync.Mutex
	fps   []float64
}

func (log *fpsLog) Init() Cmd { return log.inner.Init() }

func (log *fpsLog) Update(msg Msg) (Model, Cmd) {
	if tick, ok := msg.(TickMsg); ok {
		log.mu.Lock()
		log.fps = append(log.fps, tick.FPS)
		log.mu.Unlock()
	}
	next, cmd := log.inner.Update(msg)
	log.inner = next
	if tick, ok := msg.(TickMsg); ok && cmd == nil {
		cmd = Every(tick.Period)
	}
	return log, cmd
}

func (log *fpsLog) View() Node { return log.inner.View() }

func (log *fpsLog) samples() []float64 {
	log.mu.Lock()
	defer log.mu.Unlock()
	out := make([]float64, len(log.fps))
	copy(out, log.fps)
	return out
}

func TestRunResizeFlushesWithoutTicker(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3, Period: time.Hour})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	solid, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, host, ndarray.CPU, solid) }()
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	require.NoError(t, host.Resize(image.Pt(8, 6)))
	require.Eventually(t, func() bool {
		front := host.Front()
		return front != nil && front.Rect.Dx() == 8 && front.Rect.Dy() == 6 &&
			front.RGBAAt(7, 5) == color.RGBA{255, 0, 0, 255}
	}, 200*time.Millisecond, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func TestRunTicksWhileIdle(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	solid, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	log := &fpsLog{inner: solid}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, host, ndarray.CPU, log) }()
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, len(log.samples()), 3)
}

func TestRunPointerDoesNotZeroFPS(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	solid, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	log := &fpsLog{inner: solid}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, host, ndarray.CPU, log) }()
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	type emitter interface{ Emit(window.Event) }
	em := host.(emitter)
	deadline := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(deadline) {
		em.Emit(window.Pointer{Pos: image.Pt(1, 1)})
		time.Sleep(2 * time.Millisecond)
	}
	cancel()
	require.NoError(t, <-done)
	samples := log.samples()
	require.Greater(t, len(samples), 1, "ticks continue during pointer flood")
}

type nilView struct{}

func (nilView) Init() Cmd               { return nil }
func (nilView) Update(Msg) (Model, Cmd) { return nilView{}, nil }
func (nilView) View() Node              { return nil }

func TestRunNilView(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	assert.ErrorIs(t, Run(t.Context(), host, ndarray.CPU, nilView{}), ErrView)
}
