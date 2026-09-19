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
	expr := a.Add(b).Max(ndarray.Const(0))
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
