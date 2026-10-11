package apk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNDKTriple(t *testing.T) {
	got, err := ndkTriple("arm64")
	require.NoError(t, err)
	require.Equal(t, "aarch64-linux-android", got)
	got, err = ndkTriple("arm")
	require.NoError(t, err)
	require.Equal(t, "armv7a-linux-androideabi", got)
}
