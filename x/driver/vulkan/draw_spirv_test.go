package vulkan

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawSPIRV(t *testing.T) {
	vert, frag, inkVert, inkFrag, err := drawCode(t.Context())
	require.NoError(t, err)
	for _, spv := range [][]byte{vert, frag, inkVert, inkFrag} {
		require.GreaterOrEqual(t, len(spv), 20)
		require.Zero(t, len(spv)%4)
		require.Equal(t, uint32(0x07230203), binary.LittleEndian.Uint32(spv[:4]))
	}
	again, againFrag, againInkVert, againInkFrag, err := drawCode(t.Context())
	require.NoError(t, err)
	require.Same(t, &vert[0], &again[0])
	require.Same(t, &frag[0], &againFrag[0])
	require.Same(t, &inkVert[0], &againInkVert[0])
	require.Same(t, &inkFrag[0], &againInkFrag[0])
}
