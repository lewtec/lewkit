package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClamp01(t *testing.T) {
	require.Equal(t, 0.0, Clamp01(-0.2))
	require.Equal(t, 1.0, Clamp01(1.5))
	require.Equal(t, 0.25, Clamp01(0.25))
}
