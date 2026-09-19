package ndarray

import (
	"bytes"
	"runtime/debug"
	"testing"

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
	var src [64]byte
	for i := range src {
		src[i] = byte(i + 1)
	}
	x, err := Rand(bytes.NewReader(src[:]), Shape{16})
	require.NoError(t, err)
	got, err := x.Data()
	require.NoError(t, err)
	require.Len(t, got, 16)
	require.GreaterOrEqual(t, got[0], float32(0))
	require.Less(t, got[0], float32(1))
	y, err := Rand(bytes.NewReader(src[:]), Shape{16})
	require.NoError(t, err)
	gotY, err := y.Data()
	require.NoError(t, err)
	require.Equal(t, got, gotY)
}

func TestRandInt(t *testing.T) {
	src := []byte{2, 0, 0, 0, 4, 0, 0, 0}
	x, err := RandInt(bytes.NewReader(src), Shape{2})
	require.NoError(t, err)
	require.Equal(t, I32, x.DType())
	got := mustEval(t, x)
	require.Equal(t, []float32{1, 2}, got)
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
