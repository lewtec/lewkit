package ndarray

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func opCount(k *Kernel) int {
	n := 0
	for _, node := range k.order {
		if node.kind == kindOp {
			n++
		}
	}
	return n
}

func TestOptimizeAssocInt(t *testing.T) {
	x, err := New([]int32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	expr := x.Add(Const(int32(2))).Add(Const(int32(3))).Mul(Const(int32(4)))
	k, err := compile(expr.node)
	require.NoError(t, err)
	require.Equal(t, 2, opCount(k))
	src, err := k.GLSL()
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(src, "+"))
	require.Equal(t, 1, strings.Count(src, "*"))
	require.Contains(t, src, "5")
	require.Equal(t, []int32{24, 28, 32}, mustEval(t, expr))
}

func TestOptimizeFloatAssocStays(t *testing.T) {
	x, err := New([]float32{1, 2}, Shape{2})
	require.NoError(t, err)
	expr := x.Add(Const(float32(2))).Add(Const(float32(3)))
	k, err := compile(expr.node)
	require.NoError(t, err)
	require.Equal(t, 2, opCount(k))
	require.Equal(t, []float32{6, 7}, mustEval(t, expr))
}

func TestOptimizeDeterministic(t *testing.T) {
	build := func() string {
		x, err := New([]int32{4, 5}, Shape{2})
		require.NoError(t, err)
		expr := Const(int32(2)).Add(x).Add(Const(int32(3))).Neg().Neg()
		k, err := compile(expr.node)
		require.NoError(t, err)
		src, err := k.GLSL()
		require.NoError(t, err)
		return src
	}
	require.Equal(t, build(), build())
}

func TestOptimizeIdentities(t *testing.T) {
	x, err := New([]int32{3, 4, 5}, Shape{3})
	require.NoError(t, err)
	cases := []struct {
		name string
		expr *Tensor[int32]
	}{
		{name: "div1", expr: x.IDiv(Const(int32(1)))},
		{name: "shl0", expr: x.Shl(Const(int32(0)))},
		{name: "or0", expr: x.Or(Const(int32(0)))},
		{name: "xor0", expr: Const(int32(0)).Xor(x)},
		{name: "and-1", expr: x.And(Const(int32(-1)))},
		{name: "negneg", expr: x.Neg().Neg()},
		{name: "where-same", expr: x.CmpNe(Const(int32(0))).Where(x, x)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, err := compile(tc.expr.node)
			require.NoError(t, err)
			require.Zero(t, opCount(k))
			require.Equal(t, []int32{3, 4, 5}, mustEval(t, tc.expr))
		})
	}
}
