package d3d12

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShaderSourceKeepsTheFillContract(t *testing.T) {
	require.Contains(t, ShaderFill, "fill_vert")
	require.Contains(t, ShaderFill, "fill_frag")
	require.Contains(t, ShaderInk, "ink_frag")
	require.Contains(t, ShaderClear, "clear_frag")
	require.Contains(t, ShaderFill, "swapRB")
	require.Contains(t, ShaderFill, "clipH")
	require.Contains(t, ShaderFill, "1.0 - pos.y / extentY * 2.0")
	require.Contains(t, ShaderClear, "1.0 - unit.y * 2.0")
}
