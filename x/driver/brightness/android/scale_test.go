package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrightnessFraction(t *testing.T) {
	got, err := brightnessFraction(0, 0, 255)
	require.NoError(t, err)
	require.Zero(t, got)

	got, err = brightnessFraction(255, 0, 255)
	require.NoError(t, err)
	require.InDelta(t, 1, got, 0.0001)

	got, err = brightnessFraction(10, 10, 110)
	require.NoError(t, err)
	require.Zero(t, got)

	got, err = brightnessFraction(60, 10, 110)
	require.NoError(t, err)
	require.InDelta(t, 0.5, got, 0.0001)

	_, err = brightnessFraction(1, 5, 5)
	require.ErrorIs(t, err, errScale)
}
