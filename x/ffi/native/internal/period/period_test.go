package period

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytes(t *testing.T) {
	require.Equal(t, 3840, Bytes(48000, 4))
	require.Equal(t, 8, Bytes(1, 8))
	require.Equal(t, 4, Bytes(0, 4))
	require.Equal(t, 6, Bytes(80, 6))
}
