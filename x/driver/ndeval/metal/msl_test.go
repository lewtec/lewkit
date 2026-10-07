package metal

import (
	"math"
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
	msl, threads, err := mslSource(expr.Kernel())
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
	msl, _, err := mslSource(view.Kernel())
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
	msl, _, err := mslSource(perm.Kernel())
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
	msl, _, err = mslSource(quot.Kernel())
	require.NoError(t, err)
	require.Contains(t, msl, "device int*")
	require.True(t, strings.Contains(msl, "/"))
}

func TestTranslateU8(t *testing.T) {
	raw, err := ndarray.New([]uint8{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	require.NoError(t, raw.Resize(raw.Shape()))
	msl, _, err := mslSource(raw.Kernel())
	require.NoError(t, err)
	require.Contains(t, msl, "device uint* o [[buffer(0)]]")
	require.Contains(t, msl, "device uint* x1 [[buffer(1)]]")
}

func TestTranslateInf(t *testing.T) {
	low, err := ndarray.Full(float32(math.Inf(-1)), ndarray.Shape{1})
	require.NoError(t, err)
	require.NoError(t, low.Resize(low.Shape()))
	msl, _, err := mslSource(low.Kernel())
	require.NoError(t, err)
	require.Contains(t, msl, "as_type<float>(")
	require.NotContains(t, msl, "uintBitsToFloat")
}
