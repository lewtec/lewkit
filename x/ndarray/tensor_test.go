package ndarray

import (
	"runtime/debug"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestZerosOnes(t *testing.T) {
	z, err := Zeros(Shape{2, 3})
	require.NoError(t, err)
	require.Equal(t, Shape{2, 3}, z.Shape())
	require.Equal(t, []float32{0, 0, 0, 0, 0, 0}, mustEval(t, z))

	o, err := Ones(Shape{4})
	require.NoError(t, err)
	require.Equal(t, []float32{1, 1, 1, 1}, mustEval(t, o))
}

func TestFull(t *testing.T) {
	x, err := Full(3, Shape{2, 2})
	require.NoError(t, err)
	require.Equal(t, []float32{3, 3, 3, 3}, mustEval(t, x))
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
	require.Equal(t, []float32{2, 3, 4}, mustEval(t, a.Add(b)))
}

func TestTensorAddSame(t *testing.T) {
	a, err := New([]float32{1, 2, 3}, Shape{3})
	require.NoError(t, err)
	require.Equal(t, []float32{2, 4, 6}, mustEval(t, a.Add(a)))
}

func TestTensorAddConst(t *testing.T) {
	a, err := Ones(Shape{3})
	require.NoError(t, err)
	b, err := Full(2, Shape{3})
	require.NoError(t, err)
	require.Equal(t, []float32{3, 3, 3}, mustEval(t, a.Add(b)))
}

func TestTensorShapeMismatch(t *testing.T) {
	a, err := Ones(Shape{2})
	require.NoError(t, err)
	b, err := Ones(Shape{3})
	require.NoError(t, err)
	require.ErrorIs(t, a.Add(b).Eval(t.Context(), CPU, nil), ErrShape)
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
	dst := make([]float32, out.Size())
	require.NoError(t, out.Eval(t.Context(), &Vulkan{Device: d}, dst))
	require.Equal(t, []float32{0, 3, 0, 5}, dst)
}

func TestTensorExecOnes(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	x, err := Ones(Shape{8})
	require.NoError(t, err)
	dst := make([]float32, x.Size())
	require.NoError(t, x.Eval(t.Context(), &Vulkan{Device: d}, dst))
	require.Equal(t, []float32{1, 1, 1, 1, 1, 1, 1, 1}, dst)
}

func TestTensorExecNilDevice(t *testing.T) {
	x, err := Ones(Shape{2})
	require.NoError(t, err)
	require.ErrorIs(t, x.Eval(t.Context(), &Vulkan{}, make([]float32, 2)), ErrOp)
}

func TestTensorEvalAllocs(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4}, Shape{4})
	require.NoError(t, err)
	b, err := Ones(Shape{4})
	require.NoError(t, err)
	out := a.Add(b)
	dst := make([]float32, out.Size())
	require.NoError(t, out.Eval(t.Context(), CPU, dst))
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	n := testing.AllocsPerRun(50, func() {
		if err := out.Eval(t.Context(), CPU, dst); err != nil {
			panic(err)
		}
	})
	require.Equal(t, 0.0, n)
}
