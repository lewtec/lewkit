package nn

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
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
	require.NoError(t, y.Eval(t.Context(), ndarray.CPU))
	got, err := y.Data()
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, ndarray.Shape{2}, y.Shape())

	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	test.CloseOnCleanup(t, y)
	require.NoError(t, y.Eval(t.Context(), &ndarray.Vulkan{Device: d}))
	gpu, err := y.Data()
	require.NoError(t, err)
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
