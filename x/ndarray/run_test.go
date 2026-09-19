package ndarray_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func mustEvalCPU(t *testing.T, x *ndarray.Tensor) []float32 {
	t.Helper()
	dst := make([]float32, x.Size())
	require.NoError(t, x.Eval(t.Context(), ndarray.CPU, dst))
	return dst
}

func TestExecAdd(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	a, err := ndarray.New([]float32{-1, 2, -3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{2, -1, 5, -1}, ndarray.Shape{4})
	require.NoError(t, err)
	expr := a.Add(b).Max(ndarray.Const(0))
	want := mustEvalCPU(t, expr)
	got := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), ndeval.New(d), got))
	require.Equal(t, want, got)
}

func TestExecPermute(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	a, err := ndarray.New([]float32{1, 2, 3, 4, 5, 6}, ndarray.Shape{3, 2})
	require.NoError(t, err)
	raw, err := ndarray.New([]float32{10, 20, 30, 40, 50, 60}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	b, err := raw.Permute(1, 0)
	require.NoError(t, err)
	expr := a.Add(b)
	want := mustEvalCPU(t, expr)
	got := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), ndeval.New(d), got))
	require.Equal(t, want, got)
}
