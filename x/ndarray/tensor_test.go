package ndarray

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestZerosOnes(t *testing.T) {
	z, err := Zeros(Shape{2, 3})
	require.NoError(t, err)
	require.Equal(t, Shape{2, 3}, z.Shape())
	require.NoError(t, z.Eval(t.Context(), CPU))
	got, err := z.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{0, 0, 0, 0, 0, 0}, got)

	o, err := Ones(Shape{4})
	require.NoError(t, err)
	require.NoError(t, o.Eval(t.Context(), CPU))
	got, err = o.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{1, 1, 1, 1}, got)
}

func TestFull(t *testing.T) {
	x, err := Full(3, Shape{2, 2})
	require.NoError(t, err)
	require.NoError(t, x.Eval(t.Context(), CPU))
	got, err := x.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{3, 3, 3, 3}, got)
}

func TestRand(t *testing.T) {
	x, err := Rand(Shape{16})
	require.NoError(t, err)
	got, err := x.Data()
	require.NoError(t, err)
	require.Len(t, got, 16)
	any := false
	for _, v := range got {
		require.GreaterOrEqual(t, v, float32(0))
		require.Less(t, v, float32(1))
		if v > 0 {
			any = true
		}
	}
	require.True(t, any)
}

func TestNew(t *testing.T) {
	x, err := New([]float32{1, 2, 3, 4}, Shape{2, 2})
	require.NoError(t, err)
	got, err := x.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{1, 2, 3, 4}, got)
	_, err = New([]float32{1}, Shape{2})
	require.ErrorIs(t, err, ErrSize)
}

func TestTensorAdd(t *testing.T) {
	a, err := New([]float32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	b, err := Ones(Shape{3})
	require.NoError(t, err)
	sum := a.Add(b)
	require.NoError(t, sum.Eval(t.Context(), CPU))
	got, err := sum.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{2, 3, 4}, got)
}

func TestTensorAddSame(t *testing.T) {
	a, err := New([]float32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	sum := a.Add(a)
	require.NoError(t, sum.Eval(t.Context(), CPU))
	got, err := sum.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{2, 4, 6}, got)
}

func TestTensorAddConst(t *testing.T) {
	a, err := Ones(Shape{3})
	require.NoError(t, err)
	b, err := Full(2, Shape{3})
	require.NoError(t, err)
	sum := a.Add(b)
	require.NoError(t, sum.Eval(t.Context(), CPU))
	got, err := sum.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{3, 3, 3}, got)
}

func TestTensorShapeMismatch(t *testing.T) {
	a, err := Ones(Shape{2})
	require.NoError(t, err)
	b, err := Ones(Shape{3})
	require.NoError(t, err)
	require.ErrorIs(t, a.Add(b).Eval(t.Context(), CPU), ErrShape)
}

func TestTensorExec(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	a, err := New([]float32{-1, 2, -3, 4}, Shape{4})
	require.NoError(t, err)
	b, err := Ones(Shape{4})
	require.NoError(t, err)
	zero, err := Full(0, Shape{4})
	require.NoError(t, err)
	out := a.Add(b).Max(zero)
	require.NoError(t, out.Eval(t.Context(), &Vulkan{Device: d}))
	got, err := out.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{0, 3, 0, 5}, got)
}

func TestTensorExecOnes(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	x, err := Ones(Shape{8})
	require.NoError(t, err)
	require.NoError(t, x.Eval(t.Context(), &Vulkan{Device: d}))
	got, err := x.Data()
	require.NoError(t, err)
	require.Equal(t, []float32{1, 1, 1, 1, 1, 1, 1, 1}, got)
}

func TestTensorExecNilDevice(t *testing.T) {
	x, err := Ones(Shape{2})
	require.NoError(t, err)
	require.ErrorIs(t, x.Eval(t.Context(), &Vulkan{}), ErrOp)
}
