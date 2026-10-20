package glsl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSPIRV(t *testing.T) {
	require.False(t, IsSPIRV(nil))
	require.False(t, IsSPIRV([]byte("#version 450\n")))
	magic := []byte{0x03, 0x02, 0x23, 0x07}
	require.True(t, IsSPIRV(magic))
}

func TestCompileStageRejects(t *testing.T) {
	_, err := CompileStage(t.Context(), 3, []byte("void main() {}\n"))
	require.Error(t, err)
}

func TestCompileStageMiss(t *testing.T) {
	src := []byte("#version 450\nvoid main() {}\n")
	_, err := CompileStage(t.Context(), StageCompute, src)
	require.ErrorIs(t, err, ErrCompile)
}

func TestLoadSPIRVPassthrough(t *testing.T) {
	spv := []byte{0x03, 0x02, 0x23, 0x07, 0, 0, 0, 0}
	again, err := Load(t.Context(), spv)
	require.NoError(t, err)
	require.Equal(t, spv, again)
}
