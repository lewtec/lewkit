package ndarray

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func mustTracker(t *testing.T, shape Shape) Tracker {
	t.Helper()
	tracker, err := Of(shape)
	require.NoError(t, err)
	return tracker
}

func mustEval(t *testing.T, x *Tensor) []float32 {
	t.Helper()
	dst := make([]float32, x.Size())
	require.NoError(t, x.Eval(t.Context(), CPU, dst))
	return dst
}

func TestCompileOneMain(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4, 5, 6}, Shape{2, 3})
	require.NoError(t, err)
	b, err := New([]float32{1, 1, 1, 1, 1, 1}, Shape{2, 3})
	require.NoError(t, err)
	k, err := compile(a.Add(b).Mul(Const(2)).Max(Const(0)).node)
	require.NoError(t, err)
	src := k.GLSL()
	require.Equal(t, 1, strings.Count(src, "void main()"))
	require.Equal(t, 1, strings.Count(src, "gl_GlobalInvocationID"))
	require.Equal(t, Shape{2, 3}, k.Shape())
	require.Equal(t, 3, k.Bindings())
}

func TestEvalAdd(t *testing.T) {
	a, err := New([]float32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	b, err := New([]float32{4, 5, 6}, Shape{3})
	require.NoError(t, err)
	require.Equal(t, []float32{5, 7, 9}, mustEval(t, a.Add(b)))
}

func TestEvalPermuteAdd(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4, 5, 6}, Shape{3, 2})
	require.NoError(t, err)
	raw, err := New([]float32{10, 20, 30, 40, 50, 60}, Shape{2, 3})
	require.NoError(t, err)
	b, err := raw.Permute(1, 0)
	require.NoError(t, err)
	require.Equal(t, []float32{11, 42, 23, 54, 35, 66}, mustEval(t, a.Add(b)))
}

func TestEvalPad(t *testing.T) {
	a, err := New([]float32{4, 5}, Shape{2})
	require.NoError(t, err)
	padded, err := a.Pad([][2]int{{1, 1}})
	require.NoError(t, err)
	require.Equal(t, []float32{1, 5, 6, 1}, mustEval(t, padded.Add(Const(1))))
}

func TestEvalRelu(t *testing.T) {
	a, err := New([]float32{-2, 0, 3, -0.5}, Shape{4})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 0, 3, 0}, mustEval(t, a.Max(Const(0))))
}

func TestEvalWhere(t *testing.T) {
	a, err := New([]float32{-1, 2, -3}, Shape{3})
	require.NoError(t, err)
	got := mustEval(t, a.CmpLt(Const(0)).Where(Const(0), a))
	require.Equal(t, []float32{0, 2, 0}, got)
}

func TestCompileShapeMismatch(t *testing.T) {
	a, err := New([]float32{1, 2}, Shape{2})
	require.NoError(t, err)
	b, err := New([]float32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	require.ErrorIs(t, a.Add(b).Eval(t.Context(), CPU, nil), ErrShape)
}

func TestCompileGLSL(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4}, Shape{2, 2})
	require.NoError(t, err)
	b, err := New([]float32{1, 1, 1, 1}, Shape{2, 2})
	require.NoError(t, err)
	k, err := compile(a.Add(b).Mul(Const(0.5)).node)
	require.NoError(t, err)
	spirv, err := k.SPIRV(t.Context())
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(spirv))
}

func TestRealSize(t *testing.T) {
	require.Equal(t, 6, mustTracker(t, Shape{2, 3}).RealSize())
	require.Equal(t, 1, mustTracker(t, Shape{}).RealSize())
	ex, err := mustTracker(t, Shape{1, 3}).Expand(Shape{4, 3})
	require.NoError(t, err)
	require.Equal(t, 3, ex.RealSize())
	p, err := mustTracker(t, Shape{2}).Pad([][2]int{{1, 1}})
	require.NoError(t, err)
	require.Equal(t, 2, p.RealSize())
}
