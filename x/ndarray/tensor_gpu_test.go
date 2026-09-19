package ndarray_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestTensorExec(t *testing.T) {
	evaluator := mustGPU(t)
	a, err := ndarray.New([]float32{-1, 2, -3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.Ones[float32](ndarray.Shape{4})
	require.NoError(t, err)
	zero, err := ndarray.Full(float32(0), ndarray.Shape{4})
	require.NoError(t, err)
	out := a.Add(b).Max(zero)
	dst := make([]float32, out.Size())
	require.NoError(t, out.Eval(t.Context(), evaluator, dst))
	require.Equal(t, []float32{0, 3, 0, 5}, dst)
}

func TestTensorExecOnes(t *testing.T) {
	evaluator := mustGPU(t)
	x, err := ndarray.Ones[float32](ndarray.Shape{8})
	require.NoError(t, err)
	dst := make([]float32, x.Size())
	require.NoError(t, x.Eval(t.Context(), evaluator, dst))
	require.Equal(t, []float32{1, 1, 1, 1, 1, 1, 1, 1}, dst)
}

func TestTensorExecLoop(t *testing.T) {
	evaluator := mustGPU(t)
	a, err := ndarray.New([]float32{1, 2, 3, 4, 5, 6, 7, 8}, ndarray.Shape{8})
	require.NoError(t, err)
	out := a.Add(ndarray.Const(float32(1)))
	dst := make([]float32, 8)
	for range 20 {
		require.NoError(t, out.Eval(t.Context(), evaluator, dst))
	}
	require.Equal(t, []float32{2, 3, 4, 5, 6, 7, 8, 9}, dst)
}
