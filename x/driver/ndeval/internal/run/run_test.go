package run

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	"github.com/lewtec/lewkit/x/ndarray"
)

type mem struct {
	body     []byte
	closeErr error
	closed   bool
}

func (m *mem) Len() int { return len(m.body) }

func (m *mem) Store(data []byte) error {
	m.body = append([]byte(nil), data...)
	return nil
}

func (m *mem) Read(dst []byte) error {
	copy(dst, m.body)
	return nil
}

func (m *mem) Close() error {
	m.closed = true
	return m.closeErr
}

func kernel(t *testing.T) *ndarray.Kernel {
	t.Helper()
	left, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	right, err := ndarray.New([]float32{5, 6, 7, 8}, ndarray.Shape{4})
	require.NoError(t, err)
	sum := left.Add(right)
	require.NoError(t, sum.Resize(sum.Shape()))
	require.NotNil(t, sum.Kernel())
	return sum.Kernel()
}

func TestEvalRejectsACancelledContextBeforeTheDevice(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Eval[*mem](ctx, true, nil, false, nil, false, nil, nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestEvalSkipsTheContextWhenTheBackendDoes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Eval[*mem](ctx, false, nil, false, nil, false, nil, nil, nil)
	require.ErrorIs(t, err, ndarray.ErrOp)
}

func TestEvalRejectsAMissingDeviceBeforeTheLock(t *testing.T) {
	k := kernel(t)
	var mu sync.Mutex
	mu.Lock()
	done := make(chan error, 1)
	go func() {
		done <- Eval(t.Context(), true, k, false, &mu, true, &slots.Set[*mem]{}, make([]byte, k.OutputBytes()), func() error {
			return errors.New("dispatched")
		})
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, ndarray.ErrOp)
	case <-time.After(time.Second):
		require.FailNow(t, "eval locked before the device check")
	}
	mu.Unlock()
}

func TestEvalRejectsAMissingPipelineUnderTheLock(t *testing.T) {
	k := kernel(t)
	var ran bool
	err := Eval(t.Context(), true, k, true, &sync.Mutex{}, false, &slots.Set[*mem]{}, make([]byte, k.OutputBytes()), func() error {
		ran = true
		return nil
	})
	require.ErrorIs(t, err, ndarray.ErrOp)
	require.False(t, ran)
}

func TestEvalRejectsAShortOutput(t *testing.T) {
	k := kernel(t)
	var ran bool
	err := Eval(t.Context(), true, k, true, nil, true, &slots.Set[*mem]{}, nil, func() error {
		ran = true
		return nil
	})
	require.ErrorIs(t, err, ndarray.ErrSize)
	require.False(t, ran)
}

func TestEvalReadsTheHostResult(t *testing.T) {
	k := kernel(t)
	body := make([]byte, k.OutputBytes())
	for i := range body {
		body[i] = byte(i + 1)
	}
	set := &slots.Set[*mem]{Output: &mem{body: body}}
	got := make([]byte, k.OutputBytes())
	err := Eval(t.Context(), true, k, true, &sync.Mutex{}, true, set, got, func() error {
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, body, got)
}

func TestGroupsPassesThePush(t *testing.T) {
	k := kernel(t)
	set := &slots.Set[*mem]{}
	var groups int
	err := Groups(func() error { return nil }, set, k, func(_ []*mem, push []byte, n int) error {
		groups = n
		require.Len(t, push, ndarray.PushBytes)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, groups)
}

func TestGroupsKeepsTheOpenGLCount(t *testing.T) {
	k := kernel(t)
	set := &slots.Set[*mem]{}
	var groups uint32
	err := Groups(func() error { return nil }, set, k, func(_ []*mem, push []byte, n uint32) error {
		groups = n
		require.Len(t, push, ndarray.PushBytes)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), groups)
}

func TestGroupsStopsWhenPrepareFails(t *testing.T) {
	k := kernel(t)
	prep := errors.New("fit")
	err := Groups(func() error { return prep }, &slots.Set[*mem]{}, k, func([]*mem, []byte, int) error {
		return errors.New("ran")
	})
	require.ErrorIs(t, err, prep)
}

func TestCloseJoinsThePipelineAndTheBuffers(t *testing.T) {
	bufErr := errors.New("buf")
	pipeErr := errors.New("pipe")
	buf := &mem{closeErr: bufErr}
	set := &slots.Set[*mem]{Output: buf}
	var forgot int
	forget := func() { forgot++ }
	err := Close(&sync.Mutex{}, &forget, func() error { return pipeErr }, set)
	require.ErrorIs(t, err, pipeErr)
	require.ErrorIs(t, err, bufErr)
	require.Equal(t, 1, forgot)
	require.Nil(t, forget)
	require.True(t, buf.closed)
	err = Close(nil, &forget, func() error { return nil }, set)
	require.NoError(t, err)
	require.Equal(t, 1, forgot)
}
