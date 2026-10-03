package glsl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashSeparatesStage(t *testing.T) {
	src := []byte("void main() {}\n")
	require.Equal(t, Hash(StageVertex, src), Hash(StageVertex, src))
	require.NotEqual(t, Hash(StageVertex, src), Hash(StageFragment, src))
	require.NotEqual(t, Hash(StageVertex, src), Hash(StageVertex, []byte("other")))
}

func TestRegisterHash(t *testing.T) {
	src := []byte("registry-same")
	sum := Hash(StageCompute, src)
	spirv := sampleSPIRV(0x11)
	require.NoError(t, RegisterHash(StageCompute, sum, spirv))
	require.NoError(t, RegisterHash(StageCompute, sum, spirv))
	require.ErrorIs(t, RegisterHash(StageCompute, sum, sampleSPIRV(0x22)), ErrExist)
	require.ErrorIs(t, RegisterHash(9, sum, spirv), ErrCompile)
	require.ErrorIs(t, RegisterHash(StageCompute, sum, []byte{0x03, 0x02, 0x23, 0x07}), ErrCompile)

	got, ok := Lookup(StageCompute, src)
	require.True(t, ok)
	require.Equal(t, spirv, got)
	got[4] = 0xff
	again, ok := Lookup(StageCompute, src)
	require.True(t, ok)
	require.Equal(t, spirv, again)
	_, ok = Lookup(StageVertex, src)
	require.False(t, ok)
}

func TestCompileStageUsesRegistry(t *testing.T) {
	src := []byte("registry-skips-glslang")
	want := sampleSPIRV(0x31)
	require.NoError(t, RegisterHash(StageVertex, Hash(StageVertex, src), want))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := CompileStage(ctx, StageVertex, src)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestRegisterHashKeepsSlice(t *testing.T) {
	src := []byte("registry-keeps-slice")
	spirv := sampleSPIRV(0x41)
	require.NoError(t, RegisterHash(StageCompute, Hash(StageCompute, src), spirv))
	spirv[4] = 0x42
	got, ok := Lookup(StageCompute, src)
	require.True(t, ok)
	require.Equal(t, byte(0x42), got[4])
}

func TestMustRegisterHashPanics(t *testing.T) {
	require.Panics(t, func() {
		MustRegisterHash(StageVertex, Hash(StageVertex, []byte("bad")), []byte{1, 2, 3, 4})
	})
}

func sampleSPIRV(mark byte) []byte {
	spv := []byte{
		0x03, 0x02, 0x23, 0x07,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	spv[4] = mark
	return spv
}
