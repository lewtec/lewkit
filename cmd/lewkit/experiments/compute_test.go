package experiments

import (
	"encoding/binary"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
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

func TestLoadThenDispatch(t *testing.T) {
	src := []byte(`#version 450
layout(local_size_x = 1) in;
layout(set = 0, binding = 0) buffer Data { uint v; } data;
void main() { data.v = 2u; }
`)
	spirv, err := glsl.Load(t.Context(), src)
	require.NoError(t, err)
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.NoError(t, buf.Write(make([]byte, 4)))
	sh, err := d.Shader(t.Context(), spirv, 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	require.NoError(t, d.Run(sh, 1, 1, 1, buf))
	got := make([]byte, 4)
	require.NoError(t, buf.Read(got))
	require.Equal(t, uint32(2), binary.LittleEndian.Uint32(got))
}
