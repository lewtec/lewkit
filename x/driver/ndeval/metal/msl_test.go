package metal

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestTranslateAdd(t *testing.T) {
	a, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{4, 3, 2, 1}, ndarray.Shape{4})
	require.NoError(t, err)
	expr := a.Add(b)
	require.NoError(t, expr.Resize(expr.Shape()))
	src, err := expr.Kernel().GLSL()
	require.NoError(t, err)
	msl, threads, err := mslSource(src)
	require.NoError(t, err)
	require.Equal(t, 128, threads)
	require.Contains(t, msl, "kernel void ndeval(")
	require.Contains(t, msl, "device float* o [[buffer(0)]]")
	require.Contains(t, msl, "device float* x1 [[buffer(1)]]")
	require.Contains(t, msl, "device float* x2 [[buffer(2)]]")
	require.Contains(t, msl, "constant Push& push [[buffer(3)]]")
	require.Contains(t, msl, "uint gi = gid;")
	require.NotContains(t, msl, "gl_")
	require.NotContains(t, msl, "layout(")
	require.Contains(t, msl, "+")
}

func TestTranslateBackdropCast(t *testing.T) {
	shape := ndarray.Shape{1, 1, 4}
	channel := ndarray.Coord(2, shape)
	turn := ndarray.Const(float32(0.25))
	expr := channel.Equal(ndarray.Const(int32(0))).Where(
		turn.Sin().Mul(ndarray.Const(float32(255))),
		ndarray.Const(float32(0)),
	)
	view := expr.Cast[uint8]()
	require.NoError(t, view.Resize(ndarray.Shape{2, 2, 4}))
	src, err := view.Kernel().GLSL()
	require.NoError(t, err)
	msl, _, err := mslSource(src)
	require.NoError(t, err)
	require.Contains(t, msl, "device uint* o [[buffer(0)]]")
	require.Contains(t, msl, "uint gi = gid;")
	require.NotContains(t, msl, "gl_")
	require.NotContains(t, msl, "layout(")
}

func TestTranslatePermuteAndInt(t *testing.T) {
	raw, err := ndarray.New([]int32{1, 2, 3, 4, 5, 6}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	perm, err := raw.Permute(1, 0)
	require.NoError(t, err)
	require.NoError(t, perm.Resize(perm.Shape()))
	src, err := perm.Kernel().GLSL()
	require.NoError(t, err)
	msl, _, err := mslSource(src)
	require.NoError(t, err)
	require.Contains(t, msl, "device int* o [[buffer(0)]]")
	require.Contains(t, msl, "device int* x1 [[buffer(1)]]")
	require.NotContains(t, msl, "gl_")

	numerator, err := ndarray.New([]int32{5, 9}, ndarray.Shape{2})
	require.NoError(t, err)
	divisor, err := ndarray.New([]int32{2, 0}, ndarray.Shape{2})
	require.NoError(t, err)
	quot := numerator.IDiv(divisor)
	require.NoError(t, quot.Resize(quot.Shape()))
	src, err = quot.Kernel().GLSL()
	require.NoError(t, err)
	msl, _, err = mslSource(src)
	require.NoError(t, err)
	require.Contains(t, msl, "device int*")
	require.True(t, strings.Contains(msl, "/"))
}

func TestTranslateU8(t *testing.T) {
	raw, err := ndarray.New([]uint8{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	require.NoError(t, raw.Resize(raw.Shape()))
	src, err := raw.Kernel().GLSL()
	require.NoError(t, err)
	msl, _, err := mslSource(src)
	require.NoError(t, err)
	require.Contains(t, msl, "device uint* o [[buffer(0)]]")
	require.Contains(t, msl, "device uint* x1 [[buffer(1)]]")
}

func TestTranslateRejectsForeignGLSL(t *testing.T) {
	_, _, err := mslSource("#version 450\nvoid main() {}\n")
	require.ErrorIs(t, err, ndarray.ErrOp)
}
