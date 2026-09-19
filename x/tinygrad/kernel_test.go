package tinygrad

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func st(t *testing.T, shape ...int) Tracker {
	t.Helper()
	tr, err := Of(shape...)
	require.NoError(t, err)
	return tr
}

func TestCompileOneMain(t *testing.T) {
	a := In(0, st(t, 2, 3))
	b := In(1, st(t, 2, 3))
	k, err := Compile(a.Add(b).Mul(Const(2)).Max(Const(0)))
	require.NoError(t, err)
	src := k.GLSL()
	require.Equal(t, 1, strings.Count(src, "void main()"))
	require.Equal(t, 1, strings.Count(src, "gl_GlobalInvocationID"))
	require.Equal(t, []int{2, 3}, k.Shape())
	require.Equal(t, 3, k.Bindings())
	require.Equal(t, []int{0, 1}, k.Slots())
}

func TestEvalAdd(t *testing.T) {
	k, err := Compile(In(0, st(t, 3)).Add(In(1, st(t, 3))))
	require.NoError(t, err)
	got, err := k.Eval([]float32{1, 2, 3}, []float32{4, 5, 6})
	require.NoError(t, err)
	require.Equal(t, []float32{5, 7, 9}, got)
}

func TestEvalPermuteAdd(t *testing.T) {
	perm, err := st(t, 2, 3).Permute(1, 0)
	require.NoError(t, err)
	// A is (3,2) row-major. B is stored as (2,3) and viewed as (3,2) via permute.
	k, err := Compile(In(0, st(t, 3, 2)).Add(In(1, perm)))
	require.NoError(t, err)
	a := []float32{1, 2, 3, 4, 5, 6}       // (3,2)
	b := []float32{10, 20, 30, 40, 50, 60} // (2,3) [10 20 30 / 40 50 60]
	// perm (1,0) of (2,3) reads as (3,2): 10 40 / 20 50 / 30 60
	got, err := k.Eval(a, b)
	require.NoError(t, err)
	require.Equal(t, []float32{11, 42, 23, 54, 35, 66}, got)
}

func TestEvalPad(t *testing.T) {
	padded, err := st(t, 2).Pad([][2]int{{1, 1}})
	require.NoError(t, err)
	k, err := Compile(In(0, padded).Add(Const(1)))
	require.NoError(t, err)
	got, err := k.Eval([]float32{4, 5})
	require.NoError(t, err)
	require.Equal(t, []float32{1, 5, 6, 1}, got)
}

func TestEvalRelu(t *testing.T) {
	k, err := Compile(In(0, st(t, 4)).Max(Const(0)))
	require.NoError(t, err)
	got, err := k.Eval([]float32{-2, 0, 3, -0.5})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 0, 3, 0}, got)
}

func TestEvalWhere(t *testing.T) {
	p := In(0, st(t, 3)).CmpLt(Const(0))
	k, err := Compile(p.Where(Const(0), In(0, st(t, 3))))
	require.NoError(t, err)
	got, err := k.Eval([]float32{-1, 2, -3})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 2, 0}, got)
}

func TestCompileShapeMismatch(t *testing.T) {
	n := In(0, st(t, 2)).Add(In(1, st(t, 3)))
	_, err := Compile(n)
	require.ErrorIs(t, err, ErrShape)
}

func TestCompileGLSL(t *testing.T) {
	k, err := Compile(In(0, st(t, 2, 2)).Add(In(1, st(t, 2, 2))).Mul(Const(0.5)))
	require.NoError(t, err)
	spv, err := k.SPIRV(t.Context())
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spv))
}

func TestRealSize(t *testing.T) {
	require.Equal(t, 6, st(t, 2, 3).RealSize())
	require.Equal(t, 1, st(t).RealSize())
	ex, err := st(t, 1, 3).Expand(4, 3)
	require.NoError(t, err)
	require.Equal(t, 3, ex.RealSize())
	p, err := st(t, 2).Pad([][2]int{{1, 1}})
	require.NoError(t, err)
	require.Equal(t, 2, p.RealSize())
}
