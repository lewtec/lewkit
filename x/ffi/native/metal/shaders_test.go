package metal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShaderSourceKeepsTheFillContract(t *testing.T) {
	require.Contains(t, ShaderSource, "fill_vert")
	require.Contains(t, ShaderSource, "fill_frag")
	require.Contains(t, ShaderSource, "ink_frag")
	require.Contains(t, ShaderSource, "clear_frag")
	require.Contains(t, ShaderSource, "swapRB")
	require.Contains(t, ShaderSource, "clipH")
	require.Contains(t, ShaderSource, "1.0 - pos.y / push.extentY * 2.0")
}
