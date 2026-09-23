package singleton

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s := NewSingleton(func(context.Context) (int, error) {
		require.FailNow(t, "handler must not run")
		return 0, nil
	})
	_, err := s.GetContext(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetContextWaiterCancel(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	s := NewSingleton(func(context.Context) (int, error) {
		close(started)
		<-release
		return 1, nil
	})
	done := make(chan struct{})
	go func() {
		_, _ = s.GetContext(t.Context())
		close(done)
	}()
	<-started
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := s.GetContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	close(release)
	<-done
}

func TestGetOnce(t *testing.T) {
	var n atomic.Int32
	s := NewSingleton(func(context.Context) (int, error) {
		n.Add(1)
		return 7, nil
	})
	a, err := s.GetContext(t.Context())
	require.NoError(t, err)
	b, err := s.GetContext(t.Context())
	require.NoError(t, err)
	require.Equal(t, 7, a)
	require.Equal(t, 7, b)
	require.Equal(t, int32(1), n.Load())
}

func TestNewSingletonFunc(t *testing.T) {
	var n atomic.Int32
	get := NewSingletonFunc(func(context.Context) (int, error) {
		n.Add(1)
		return 3, nil
	})
	require.Equal(t, 3, get())
	require.Equal(t, 3, get())
	require.Equal(t, int32(1), n.Load())
}
