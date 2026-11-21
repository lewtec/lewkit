package ndarray_test

import (
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func mustEvalCPU[T ndarray.Number](t *testing.T, x *ndarray.Tensor[T]) []T {
	t.Helper()
	dst := make([]T, x.Size())
	require.NoError(t, x.Eval(t.Context(), ndarray.CPU, dst))
	return dst
}

func mustGPU(t *testing.T) ndarray.Evaluator {
	t.Helper()
	if _, err := vulkan.List(t.Context()); err != nil {
		t.Skip(err)
	}
	evaluator, err := ndarray.Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	return evaluator
}

func TestExecAdd(t *testing.T) {
	evaluator := mustGPU(t)
	a, err := ndarray.New([]float32{-1, 2, -3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{2, -1, 5, -1}, ndarray.Shape{4})
	require.NoError(t, err)
	expr := a.Add(b).Max(ndarray.Const(float32(0)))
	want := mustEvalCPU(t, expr)
	got := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), evaluator, got))
	require.Equal(t, want, got)
}

func TestExecPermute(t *testing.T) {
	evaluator := mustGPU(t)
	a, err := ndarray.New([]float32{1, 2, 3, 4, 5, 6}, ndarray.Shape{3, 2})
	require.NoError(t, err)
	raw, err := ndarray.New([]float32{10, 20, 30, 40, 50, 60}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	b, err := raw.Permute(1, 0)
	require.NoError(t, err)
	expr := a.Add(b)
	want := mustEvalCPU(t, expr)
	got := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), evaluator, got))
	require.Equal(t, want, got)
}

func TestExecMatchesCPUEdges(t *testing.T) {
	evaluator := mustGPU(t)
	coord, err := ndarray.Coord(0, ndarray.Shape{3}).Pad([][2]int{{1, 1}})
	require.NoError(t, err)
	padded := coord.Add(ndarray.Const(int32(1)))
	want := mustEvalCPU(t, padded)
	got := make([]int32, padded.Size())
	require.NoError(t, padded.Eval(t.Context(), evaluator, got))
	require.Equal(t, want, got)

	numerator, err := ndarray.New([]int32{5, 16777217, -2147483647}, ndarray.Shape{3})
	require.NoError(t, err)
	divisor, err := ndarray.New([]int32{0, 1, 1}, ndarray.Shape{3})
	require.NoError(t, err)
	quotient := numerator.IDiv(divisor)
	wantQuotient := mustEvalCPU(t, quotient)
	gotQuotient := make([]int32, quotient.Size())
	require.NoError(t, quotient.Eval(t.Context(), evaluator, gotQuotient))
	require.Equal(t, wantQuotient, gotQuotient)

	leaf, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	require.NoError(t, leaf.Resize(ndarray.Shape{8}))
	wantLeaf := mustEvalCPU(t, leaf)
	gotLeaf := make([]float32, leaf.Size())
	require.NoError(t, leaf.Eval(t.Context(), evaluator, gotLeaf))
	require.Equal(t, wantLeaf, gotLeaf)
}
