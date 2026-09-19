package ndarray_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestTensorExec(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	a, err := ndarray.New([]float32{-1, 2, -3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.Ones(ndarray.Shape{4})
	require.NoError(t, err)
	zero, err := ndarray.Full(0, ndarray.Shape{4})
	require.NoError(t, err)
	out := a.Add(b).Max(zero)
	dst := make([]float32, out.Size())
	require.NoError(t, out.Eval(t.Context(), ndeval.New(d), dst))
	require.Equal(t, []float32{0, 3, 0, 5}, dst)
}

func TestTensorExecOnes(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	x, err := ndarray.Ones(ndarray.Shape{8})
	require.NoError(t, err)
	dst := make([]float32, x.Size())
	require.NoError(t, x.Eval(t.Context(), ndeval.New(d), dst))
	require.Equal(t, []float32{1, 1, 1, 1, 1, 1, 1, 1}, dst)
}

func TestTensorExecNilDevice(t *testing.T) {
	x, err := ndarray.Ones(ndarray.Shape{2})
	require.NoError(t, err)
	require.ErrorIs(t, x.Eval(t.Context(), ndeval.New(nil), make([]float32, 2)), ndarray.ErrOp)
}
