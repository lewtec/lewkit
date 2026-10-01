package vulkan

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawSPIRV(t *testing.T) {
	code, err := drawCode.GetContext(t.Context())
	require.NoError(t, err)
	for _, spv := range [][]byte{code.vert, code.frag, code.inkVert, code.inkFrag} {
		require.GreaterOrEqual(t, len(spv), 20)
		require.Zero(t, len(spv)%4)
		require.Equal(t, uint32(0x07230203), binary.LittleEndian.Uint32(spv[:4]))
	}
	again, err := drawCode.GetContext(t.Context())
	require.NoError(t, err)
	require.Same(t, &code.vert[0], &again.vert[0])
	require.Same(t, &code.frag[0], &again.frag[0])
	require.Same(t, &code.inkVert[0], &again.inkVert[0])
	require.Same(t, &code.inkFrag[0], &again.inkFrag[0])
}
