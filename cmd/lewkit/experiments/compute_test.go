package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeUsage(t *testing.T) {
	text, err := cmd.Usage[Compute]("lewkit experiments compute")
	require.NoError(t, err)
	assert.Contains(t, text, "SPIR-V")
	assert.Contains(t, text, "--width")
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
