package vulkan

import (
	"math"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestGLSLAddCompiles(t *testing.T) {
	a, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{1, 1, 1, 1}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	k, err := compileKernel(a.Add(b).Mul(ndarray.Const(float32(0.5))))
	require.NoError(t, err)
	src, err := glslSource(k)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(src, "void main()"))
	require.Equal(t, 1, strings.Count(src, "gl_GlobalInvocationID"))
	require.Contains(t, src, "local_size_x = 128")
	spirv, err := glsl.Load(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))
}

func TestGLSLLiterals(t *testing.T) {
	low, err := ndarray.Full(float32(math.Inf(-1)), ndarray.Shape{1})
	require.NoError(t, err)
	k, err := compileKernel(low)
	require.NoError(t, err)
	src, err := glslSource(k)
	require.NoError(t, err)
	require.Contains(t, src, "uintBitsToFloat(")
	require.NotContains(t, src, "Inf")
	spirv, err := glsl.Load(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))

	minInt, err := ndarray.Full(int32(math.MinInt32), ndarray.Shape{1})
	require.NoError(t, err)
	k, err = compileKernel(minInt)
	require.NoError(t, err)
	src, err = glslSource(k)
	require.NoError(t, err)
	require.Contains(t, src, "0x80000000u")
	require.Contains(t, src, "buffer Out { int o[]; }")
	spirv, err = glsl.Load(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))
}

func TestGLSLGuardsCompile(t *testing.T) {
	n, err := ndarray.New([]int32{5, 9}, ndarray.Shape{2})
	require.NoError(t, err)
	d, err := ndarray.New([]int32{2, 0}, ndarray.Shape{2})
	require.NoError(t, err)
	k, err := compileKernel(n.IDiv(d).Add(n.Shl(d)))
	require.NoError(t, err)
	src, err := glslSource(k)
	require.NoError(t, err)
	spirv, err := glsl.Load(t.Context(), []byte(src))
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))
}

func compileKernel[T ndarray.Number](x *ndarray.Tensor[T]) (*ndarray.Kernel, error) {
	if err := x.Resize(x.Shape()); err != nil {
		return nil, err
	}
	return x.Kernel(), nil
}
