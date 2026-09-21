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
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 4})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	assert.ErrorIs(t, Run(t.Context(), w, ndarray.CPU, nil), ErrModel)
}

func TestRunNilWindow(t *testing.T) {
	s, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	assert.ErrorIs(t, Run(t.Context(), nil, ndarray.CPU, s), window.ErrClosed)
}

func TestPictureFrameSig(t *testing.T) {
	s, err := NewSolid(10, 20, 30, 255)
	require.NoError(t, err)
	_ = s.View()
	a := s.frameSig()
	_ = s.View()
	assert.Equal(t, a, s.frameSig())
	s.fill = Color{1, 2, 3, 255}
	_ = s.View()
	assert.NotEqual(t, a, s.frameSig())
}

func TestSolidResize(t *testing.T) {
	s, err := NewSolid(10, 20, 30, 255)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 4}, s.View().Shape())
	next, cmd := s.Update(TickMsg{Size: image.Pt(8, 6)})
	require.Nil(t, cmd)
	require.Equal(t, ndarray.Shape{6, 8, 4}, next.View().Shape())
}

func TestRunPaintsSolid(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	s, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, w, ndarray.CPU, s) }()
	require.Eventually(t, func() bool {
		return w.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255} &&
			w.Front().RGBAAt(3, 2) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

type fpsLog struct {
	inner Model
	mu    sync.Mutex
	fps   []float64
}

func (l *fpsLog) Init() Cmd { return l.inner.Init() }

func (l *fpsLog) Update(msg Msg) (Model, Cmd) {
	if tick, ok := msg.(TickMsg); ok {
		l.mu.Lock()
		l.fps = append(l.fps, tick.FPS)
		l.mu.Unlock()
	}
	next, cmd := l.inner.Update(msg)
	l.inner = next
	if tick, ok := msg.(TickMsg); ok && cmd == nil {
		cmd = Every(tick.Period)
	}
	return l, cmd
}

func (l *fpsLog) View() *ndarray.Tensor[uint8] { return l.inner.View() }

func (l *fpsLog) frameSig() uint64 {
	if s, ok := l.inner.(interface{ frameSig() uint64 }); ok {
		return s.frameSig()
	}
	return 0
}

func (l *fpsLog) samples() []float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]float64, len(l.fps))
	copy(out, l.fps)
	return out
}

func TestRunResizeFlushesWithoutTicker(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3, Period: time.Hour})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	s, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, w, ndarray.CPU, s) }()
	require.Eventually(t, func() bool {
		return w.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	require.NoError(t, w.Resize(image.Pt(8, 6)))
	require.Eventually(t, func() bool {
		front := w.Front()
		return front != nil && front.Rect.Dx() == 8 && front.Rect.Dy() == 6 &&
			front.RGBAAt(7, 5) == color.RGBA{255, 0, 0, 255}
	}, 200*time.Millisecond, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func TestRunTicksWhileIdle(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	s, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	log := &fpsLog{inner: s}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, w, ndarray.CPU, log) }()
	require.Eventually(t, func() bool {
		return w.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, len(log.samples()), 3)
}

func TestRunPointerDoesNotZeroFPS(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	s, err := NewSolid(255, 0, 0, 255)
	require.NoError(t, err)
	log := &fpsLog{inner: s}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, w, ndarray.CPU, log) }()
	require.Eventually(t, func() bool {
		return w.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	type emitter interface{ Emit(window.Event) }
	em := w.(emitter)
	deadline := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(deadline) {
		em.Emit(window.Pointer{Pos: image.Pt(1, 1)})
		time.Sleep(2 * time.Millisecond)
	}
	cancel()
	require.NoError(t, <-done)
	samples := log.samples()
	require.NotEmpty(t, samples)
	for i, fps := range samples {
		if i == 0 {
			continue
		}
		assert.NotZero(t, fps, "TickMsg %d FPS", i)
	}
}

type nilView struct{}

func (nilView) Init() Cmd                    { return nil }
func (nilView) Update(Msg) (Model, Cmd)      { return nilView{}, nil }
func (nilView) View() *ndarray.Tensor[uint8] { return nil }

func TestRunNilView(t *testing.T) {
	w, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	test.CloseOnCleanup(t, w)
	assert.ErrorIs(t, Run(t.Context(), w, ndarray.CPU, nilView{}), ErrView)
}
