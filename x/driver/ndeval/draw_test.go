package ndeval

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func TestDrawShaders(t *testing.T) {
	vert, frag, inkVert, inkFrag, err := drawCode(t.Context())
	require.NoError(t, err)
	for _, spv := range [][]byte{vert, frag, inkVert, inkFrag} {
		require.True(t, glsl.IsSPIRV(spv))
		require.Equal(t, 0, len(spv)%4)
	}
}
