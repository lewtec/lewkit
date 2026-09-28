package android

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestDecodeConfirm(t *testing.T) {
	t.Parallel()

	ok, err := decodeConfirm(confirmYes)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = decodeConfirm(confirmNo)
	require.NoError(t, err)
	require.False(t, ok)

	_, err = decodeConfirm(confirmNoActivity)
	require.ErrorIs(t, err, driver.ErrUnavailable)

	_, err = decodeConfirm(9)
	require.ErrorIs(t, err, driver.ErrUnavailable)
}

func TestTakeConfirm(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	ok, err := takeConfirm(ctx, make(chan int))
	require.False(t, ok)
	require.ErrorIs(t, err, context.Canceled)

	ch := make(chan int, 1)
	ch <- confirmYes
	ok, err = takeConfirm(t.Context(), ch)
	require.NoError(t, err)
	require.True(t, ok)
}
