package d3d12

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
	hlsl, threads, err := hlslSource(expr.Kernel())
	require.NoError(t, err)
	require.Equal(t, 128, threads)
	require.Contains(t, hlsl, "[numthreads(128, 1, 1)]")
	require.Contains(t, hlsl, "void ndeval(uint3 tid : SV_DispatchThreadID)")
	require.Contains(t, hlsl, "RWStructuredBuffer<float> o : register(u0);")
	require.Contains(t, hlsl, "StructuredBuffer<float> x1 : register(t0);")
	require.Contains(t, hlsl, "StructuredBuffer<float> x2 : register(t1);")
	require.Contains(t, hlsl, "cbuffer Push : register(b0)")
	require.Contains(t, hlsl, "uint n;")
	require.NotContains(t, hlsl, "uint n =")
	require.Contains(t, hlsl, "uint gi = gid;")
	require.Contains(t, hlsl, "uint gid = tid.x;")
	require.NotContains(t, hlsl, "gl_")
	require.NotContains(t, hlsl, "layout(")
	require.Contains(t, hlsl, "+")
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
	hlsl, _, err := hlslSource(view.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "RWStructuredBuffer<uint> o : register(u0);")
	require.Contains(t, hlsl, "uint gi = gid;")
	require.NotContains(t, hlsl, "gl_")
	require.NotContains(t, hlsl, "layout(")
}

func TestTranslatePermuteAndInt(t *testing.T) {
	raw, err := ndarray.New([]int32{1, 2, 3, 4, 5, 6}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	perm, err := raw.Permute(1, 0)
	require.NoError(t, err)
	require.NoError(t, perm.Resize(perm.Shape()))
	hlsl, _, err := hlslSource(perm.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "RWStructuredBuffer<int> o : register(u0);")
	require.Contains(t, hlsl, "StructuredBuffer<int> x1 : register(t0);")
	require.NotContains(t, hlsl, "gl_")

	numerator, err := ndarray.New([]int32{5, 9}, ndarray.Shape{2})
	require.NoError(t, err)
	divisor, err := ndarray.New([]int32{2, 0}, ndarray.Shape{2})
	require.NoError(t, err)
	quot := numerator.IDiv(divisor)
	require.NoError(t, quot.Resize(quot.Shape()))
	hlsl, _, err = hlslSource(quot.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "RWStructuredBuffer<int>")
	require.True(t, strings.Contains(hlsl, "/"))
}

func TestTranslateU8(t *testing.T) {
	raw, err := ndarray.New([]uint8{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	require.NoError(t, raw.Resize(raw.Shape()))
	hlsl, _, err := hlslSource(raw.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "RWStructuredBuffer<uint> o : register(u0);")
	require.Contains(t, hlsl, "StructuredBuffer<uint> x1 : register(t0);")
}

func TestTranslateInf(t *testing.T) {
	low, err := ndarray.Full(float32(math.Inf(-1)), ndarray.Shape{1})
	require.NoError(t, err)
	require.NoError(t, low.Resize(low.Shape()))
	hlsl, _, err := hlslSource(low.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "asfloat(")
	require.NotContains(t, hlsl, "uintBitsToFloat")
}

func TestTranslateCompare(t *testing.T) {
	a, err := ndarray.New([]int32{1, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	b, err := ndarray.New([]int32{2, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	expr := a.CmpLt(b)
	require.NoError(t, expr.Resize(expr.Shape()))
	hlsl, _, err := hlslSource(expr.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "?1:0")
	require.NotContains(t, hlsl, "int(t2<t3)")
	require.NotContains(t, hlsl, "int(t2 < t3)")
}

func TestTranslateMinInt(t *testing.T) {
	low, err := ndarray.Full(int32(math.MinInt32), ndarray.Shape{1})
	require.NoError(t, err)
	require.NoError(t, low.Resize(low.Shape()))
	hlsl, _, err := hlslSource(low.Kernel())
	require.NoError(t, err)
	require.Contains(t, hlsl, "asint(0x80000000u)")
	require.NotContains(t, strings.ReplaceAll(hlsl, "asint(0x80000000u)", ""), "0x80000000u")
}
