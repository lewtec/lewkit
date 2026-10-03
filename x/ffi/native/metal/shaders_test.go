package metal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawEnumsMatchMetal(t *testing.T) {
	require.Equal(t, 4, primStrip)
	require.Equal(t, 0, loadDontCare)
	require.Equal(t, 1, storeStore)
	require.Equal(t, 1, blendOne)
	require.Equal(t, 5, blendOneMinus)
	require.Equal(t, 80, pixelBGRA)
	require.Equal(t, 0, channelSwap)
}

func TestShaderSourceKeepsTheFillContract(t *testing.T) {
	require.Contains(t, ShaderSource, "fill_vert")
	require.Contains(t, ShaderSource, "fill_frag")
	require.Contains(t, ShaderSource, "ink_frag")
	require.Contains(t, ShaderSource, "clear_frag")
	require.Contains(t, ShaderSource, "swapRB")
	require.Contains(t, ShaderSource, "clipH")
	require.Contains(t, ShaderSource, "1.0 - pos.y / push.extentY * 2.0")
}
