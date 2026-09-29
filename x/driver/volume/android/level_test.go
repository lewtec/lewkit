package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFraction(t *testing.T) {
	require.Equal(t, 0.0, fraction(0, 15))
	require.Equal(t, 1.0, fraction(15, 15))
	require.Equal(t, 0.0, fraction(7, 0))
	require.Equal(t, 0.5, fraction(5, 10))
}

func TestIndex(t *testing.T) {
	require.Equal(t, int32(0), index(0, 15))
	require.Equal(t, int32(15), index(1, 15))
	require.Equal(t, int32(5), index(0.5, 10))
	require.Equal(t, int32(10), index(1.4, 10))
	require.Equal(t, int32(0), index(-1, 10))
}
