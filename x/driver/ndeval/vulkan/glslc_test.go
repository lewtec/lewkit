package vulkan

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func TestCompileComputeUsesRegistry(t *testing.T) {
	src := []byte("registry-only compute shader\n")
	spirv := []byte{
		0x03, 0x02, 0x23, 0x07,
		1, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	require.NoError(t, glsl.RegisterHash(glsl.StageCompute, glsl.Hash(glsl.StageCompute, src), spirv))
	got, err := compileCompute(t.Context(), src)
	require.NoError(t, err)
	require.Equal(t, spirv, got)
}

func TestCompileComputeMissCompiles(t *testing.T) {
	src := []byte("#version 450\nlayout(local_size_x = 1) in;\nvoid main() {}\n")
	got, err := compileCompute(t.Context(), src)
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(got))
}
