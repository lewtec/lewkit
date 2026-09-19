package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/wasm/glsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeUsage(t *testing.T) {
	text, err := cmd.Usage[Compute]("lewkit experiments compute")
	require.NoError(t, err)
	assert.Contains(t, text, "GLSL")
	assert.Contains(t, text, "--width")
	assert.Contains(t, text, "--smoke")
	assert.Contains(t, text, "--bindings")
}

func TestComputeShaderOptional(t *testing.T) {
	bare := cmd.ParseOK[Compute](t)
	assert.Nil(t, bare.shader)
	with := cmd.ParseOK[Compute](t, "out.spv", "--width", "320")
	require.NotNil(t, with.shader)
	assert.Equal(t, "out.spv", with.shader.Value())
	assert.Equal(t, 320, with.width.Value())
}

func TestExampleComp(t *testing.T) {
	require.Contains(t, string(exampleComp), "local_size_x = 8")
	got, err := loadShader(t.Context(), "")
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(got))
}

func TestLoadShaderFile(t *testing.T) {
	got, err := loadShader(t.Context(), "example.comp")
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(got))
}
