package android

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func TestDecodePrompt(t *testing.T) {
	t.Parallel()

	text, err := decodePrompt("ada", promptOK)
	require.NoError(t, err)
	require.Equal(t, "ada", text)

	text, err = decodePrompt("", promptOK)
	require.NoError(t, err)
	require.Empty(t, text)

	_, err = decodePrompt("ada", promptCancel)
	require.ErrorIs(t, err, launcher.ErrCanceled)

	_, err = decodePrompt("", promptNoActivity)
	require.ErrorIs(t, err, driver.ErrUnavailable)

	_, err = decodePrompt("", 9)
	require.ErrorIs(t, err, driver.ErrUnavailable)
}

func TestTakePrompt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := takePrompt(ctx, make(chan promptReply))
	require.ErrorIs(t, err, context.Canceled)

	ch := make(chan promptReply, 1)
	ch <- promptReply{text: "ada", code: promptOK}
	text, err := takePrompt(t.Context(), ch)
	require.NoError(t, err)
	require.Equal(t, "ada", text)
}
