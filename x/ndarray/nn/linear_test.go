package nn

import (
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestLinear(t *testing.T) {
	w, err := ndarray.New([]float32{1, 2, 3, 4, 5, 6}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	x, err := ndarray.New([]float32{10, 20, 30}, ndarray.Shape{3})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{100, 200}, ndarray.Shape{2})
	require.NoError(t, err)
	y, err := Linear(w, x, b)
	require.NoError(t, err)
	want := []float32{240, 520}
	got := make([]float32, y.Size())
	require.NoError(t, y.Eval(t.Context(), ndarray.CPU, got))
	require.Equal(t, want, got)
	require.Equal(t, ndarray.Shape{2}, y.Shape())

	if _, err := vulkan.List(t.Context()); err != nil {
		t.Skip(err)
	}
	evaluator, err := ndarray.Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	test.CloseOnCleanup(t, y)
	gpu := make([]float32, y.Size())
	require.NoError(t, y.Eval(t.Context(), evaluator, gpu))
	require.Equal(t, want, gpu)
}

func TestLinearShape(t *testing.T) {
	w, err := ndarray.New([]float32{1, 2, 3, 4, 5, 6}, ndarray.Shape{2, 3})
	require.NoError(t, err)
	x, err := ndarray.New([]float32{1, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{1, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	_, err = Linear(w, x, b)
	require.ErrorIs(t, err, ndarray.ErrShape)
}
