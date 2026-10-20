package reactor

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func TestCompileCompute(t *testing.T) {
	src := `#version 450
layout(local_size_x = 1) in;
layout(set = 0, binding = 0) buffer Data { uint v; } data;
void main() { data.v = 2u; }
`
	spv, err := Compile(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spv))
	require.GreaterOrEqual(t, len(spv), 20)
	require.Equal(t, 0, len(spv)%4)
}

func TestCompileVertex(t *testing.T) {
	src := `#version 450
void main() { gl_Position = vec4(0.0); }
`
	spv, err := CompileStage(t.Context(), glsl.StageVertex, []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spv))
	require.Equal(t, 0, len(spv)%4)
}

func TestCompileCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Compile(ctx, []byte(`#version 450
layout(local_size_x = 1) in;
void main() {}
`))
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoadSPIRVPassthrough(t *testing.T) {
	src := `#version 450
layout(local_size_x = 1) in;
void main() {}
`
	spv, err := Compile(t.Context(), []byte(src))
	require.NoError(t, err)
	again, err := glsl.Load(t.Context(), spv)
	require.NoError(t, err)
	require.Equal(t, spv, again)
}
