package opengl

import (
	"math"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestGLSLMarkers(t *testing.T) {
	a, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{1, 1, 1, 1}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	k, err := compileKernel(a.Add(b).Mul(ndarray.Const(float32(0.5))))
	require.NoError(t, err)

	desktop, err := glslSource(k, false)
	require.NoError(t, err)
	require.Contains(t, desktop, "#version 430")
	require.Contains(t, desktop, "layout(local_size_x = 128)")
	require.Contains(t, desktop, "layout(std430, binding = 0) buffer Out")
	require.Contains(t, desktop, "uniform uint n;")
	require.Contains(t, desktop, "gl_GlobalInvocationID")
	require.NotContains(t, desktop, "set =")
	require.NotContains(t, desktop, "push_constant")

	es, err := glslSource(k, true)
	require.NoError(t, err)
	require.Contains(t, es, "#version 310 es")
	require.Contains(t, es, "precision highp float;")
	require.Contains(t, es, "layout(std430, binding = 1)")
	require.NotContains(t, es, "set =")
}

func TestGLSLLiterals(t *testing.T) {
	low, err := ndarray.Full(float32(math.Inf(-1)), ndarray.Shape{1})
	require.NoError(t, err)
	k, err := compileKernel(low)
	require.NoError(t, err)
	src, err := glslSource(k, false)
	require.NoError(t, err)
	require.Contains(t, src, "uintBitsToFloat(")
	require.NotContains(t, src, "Inf")

	minInt, err := ndarray.Full(int32(math.MinInt32), ndarray.Shape{1})
	require.NoError(t, err)
	k, err = compileKernel(minInt)
	require.NoError(t, err)
	src, err = glslSource(k, false)
	require.NoError(t, err)
	require.Contains(t, src, "0x80000000u")
	require.Contains(t, src, "buffer Out { int o[]; }")
}

func compileKernel[T ndarray.Number](x *ndarray.Tensor[T]) (*ndarray.Kernel, error) {
	if err := x.Resize(x.Shape()); err != nil {
		return nil, err
	}
	return x.Kernel(), nil
}
