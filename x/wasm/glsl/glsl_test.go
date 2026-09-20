package glsl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSPIRV(t *testing.T) {
	require.False(t, IsSPIRV(nil))
	require.False(t, IsSPIRV([]byte("#version 450\n")))
	magic := []byte{0x03, 0x02, 0x23, 0x07}
	require.True(t, IsSPIRV(magic))
}

func TestCompileCompute(t *testing.T) {
	src := `#version 450
layout(local_size_x = 1) in;
layout(set = 0, binding = 0) buffer Data { uint v; } data;
void main() { data.v = 2u; }
`
	spv, err := Compile(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, IsSPIRV(spv))
	require.GreaterOrEqual(t, len(spv), 20)
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
	again, err := Load(t.Context(), spv)
	require.NoError(t, err)
	require.Equal(t, spv, again)
}
