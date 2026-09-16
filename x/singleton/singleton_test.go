package singleton

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s := NewSingleton(func(context.Context) (int, error) {
		t.Fatal("handler must not run")
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
