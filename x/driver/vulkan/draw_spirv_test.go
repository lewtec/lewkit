package vulkan

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
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

func TestDrawSPIRVRegistered(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	steps := []struct {
		stage glsl.Stage
		name  string
	}{
		{glsl.StageVertex, "shader/fill.vertex.glsl"},
		{glsl.StageFragment, "shader/fill.fragment.glsl"},
		{glsl.StageVertex, "shader/ink.vertex.glsl"},
		{glsl.StageFragment, "shader/ink.fragment.glsl"},
	}
	for _, shader := range steps {
		src, err := drawShaders.ReadFile(shader.name)
		require.NoError(t, err)
		spv, err := glsl.CompileStage(ctx, shader.stage, src)
		require.NoError(t, err)
		require.True(t, glsl.IsSPIRV(spv))
	}
}
