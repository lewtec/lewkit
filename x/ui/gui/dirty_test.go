package gui

import (
	"context"
	"image"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type quiet struct {
	Dirty
	paints atomic.Int32
	n      int
}

func (q *quiet) Init() Cmd { return nil }

func (q *quiet) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(window.Pointer); ok {
		return q, nil
	}
	if _, ok := msg.(window.Key); ok {
		q.Dirty, q.n = See(q.Dirty, q.n, q.n+1)
	}
	return q, nil
}

func (q *quiet) View() Node {
	q.paints.Add(1)
	return &Box{Fill: &RGB{9, 8, 7, 255}}
}

func TestDirtySkipsUnchangedPointer(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 4, Height: 3, Period: 20 * time.Millisecond})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	model := &quiet{}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, host, ndarray.CPU, model) }()
	require.Eventually(t, func() bool {
		select {
		case err := <-done:
			t.Fatalf("run: %v", err)
		default:
		}
		return model.paints.Load() == 1
	}, time.Second, 5*time.Millisecond)
	emit(t, host, window.Pointer{Pos: image.Pt(1, 1)})
	time.Sleep(40 * time.Millisecond)
	assert.Equal(t, int32(1), model.paints.Load())
	emit(t, host, window.Key{Pressed: true, Rune: 'a'})
	require.Eventually(t, func() bool { return model.paints.Load() == 2 }, time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func emit(t *testing.T, host window.Window, event window.Event) {
	t.Helper()
	host.(interface{ Emit(window.Event) }).Emit(event)
}
