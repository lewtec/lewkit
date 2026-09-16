package future

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	f := NewFuture(ctx, func(context.Context) (int, error) {
		t.Fatal("handler must not run")
		return 0, nil
	})
	_, err := f.Get()
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, FutureCancelled, f.State())
}
