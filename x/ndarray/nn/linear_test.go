package nn

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestLinear(t *testing.T) {
	w, err := ndarray.Of(ndarray.Shape{2, 3})
	require.NoError(t, err)
	x, err := ndarray.Of(ndarray.Shape{3})
	require.NoError(t, err)
	b, err := ndarray.Of(ndarray.Shape{2})
	require.NoError(t, err)
	expr, err := Linear(w, x, b)
	require.NoError(t, err)

	W := []float32{1, 2, 3, 4, 5, 6}
	X := []float32{10, 20, 30}
	B := []float32{100, 200}
	want := []float32{240, 520}

	k, err := ndarray.Compile(expr)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{2}, k.Shape())
	require.Equal(t, []int{0, 1, 2}, k.Slots())
	require.Equal(t, 1, strings.Count(k.GLSL(), "void main()"))

	got, err := k.Eval(W, X, B)
	require.NoError(t, err)
	require.Equal(t, want, got)

	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Log(err)
		return
	}
	test.CloseOnCleanup(t, d)
	test.CloseOnCleanup(t, k)
	gpu, err := k.Exec(t.Context(), d, W, X, B)
	require.NoError(t, err)
	require.Equal(t, want, gpu)
}

func TestLinearShape(t *testing.T) {
	w, err := ndarray.Of(ndarray.Shape{2, 3})
	require.NoError(t, err)
	x, err := ndarray.Of(ndarray.Shape{2})
	require.NoError(t, err)
	b, err := ndarray.Of(ndarray.Shape{2})
	require.NoError(t, err)
	_, err = Linear(w, x, b)
	require.ErrorIs(t, err, ndarray.ErrShape)
}
