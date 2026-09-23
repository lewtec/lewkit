package ndarray

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func TestRepeatMatchesUnroll(t *testing.T) {
	init, err := Full(float32(0), Shape{4})
	require.NoError(t, err)
	params, err := New([]float32{1, 5, 2, 6, 3, 7}, Shape{6})
	require.NoError(t, err)
	got, err := Repeat(init, 3, func(index *Tensor[int32], acc *Tensor[float32]) *Tensor[float32] {
		return acc.Add(params.Gather(index))
	})
	require.NoError(t, err)
	acc := init
	for step := range 3 {
		cell, err := params.Shrink([][2]int{{step, step + 1}})
		require.NoError(t, err)
		splat, err := cell.Splat()
		require.NoError(t, err)
		acc = acc.Add(splat)
	}
	require.Equal(t, mustEval(t, acc), mustEval(t, got))
	kernel, err := compile(got.node)
	require.NoError(t, err)
	src, err := kernel.GLSL()
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(src, "for ("))
	spirv, err := glsl.Load(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))
}
