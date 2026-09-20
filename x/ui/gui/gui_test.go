package gui

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
		return w.Front().RGBAAt(0, 0) == color.RGBA{255, 0, 0, 255}
	}, time.Second, 5*time.Millisecond)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, w.Front().RGBAAt(3, 2))
	cancel()
	require.NoError(t, <-done)
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
