package webview2

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWaitForPumpsUntilTheHandlerSignals(t *testing.T) {
	done := make(chan uintptr, 1)
	pumps := 0
	got, err := waitFor(context.Background(), done, make(chan error, 1), func(context.Context) error {
		pumps++
		if pumps == 2 {
			done <- 7
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, uintptr(7), got)
	require.Equal(t, 2, pumps)
}

func TestWaitForReturnsTheHandlerError(t *testing.T) {
	failed := make(chan error, 1)
	_, err := waitFor(context.Background(), make(chan uintptr, 1), failed, func(context.Context) error {
		failed <- io.EOF
		return nil
	})
	require.ErrorIs(t, err, io.EOF)
}

func TestWaitForStopsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pumps := 0
	_, err := waitFor(ctx, make(chan uintptr), make(chan error), func(context.Context) error {
		pumps++
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, pumps)
}

func TestWaitForPrefersTheHandlerError(t *testing.T) {
	failed := make(chan error, 1)
	_, err := waitFor(context.Background(), make(chan uintptr, 1), failed, func(context.Context) error {
		failed <- io.EOF
		return io.ErrClosedPipe
	})
	require.ErrorIs(t, err, io.EOF)
}

func TestWaitForReturnsAPumpError(t *testing.T) {
	_, err := waitFor(context.Background(), make(chan uintptr), make(chan error), func(context.Context) error {
		return io.ErrClosedPipe
	})
	require.ErrorIs(t, err, io.ErrClosedPipe)
}
