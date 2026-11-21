package ndarray

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoordEval(t *testing.T) {
	require.Equal(t, []float32{0, 1, 2, 0, 1, 2}, mustEval(t, Coord(1, Shape{2, 3}).Cast[float32]()))
}

func TestEvalInto(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4, 5, 6, 7, 8}, Shape{8})
	require.NoError(t, err)
	out := a.Add(Const(float32(1)))
	dst := make([]float32, 8)
	require.NoError(t, out.Eval(t.Context(), CPU, dst))
	require.Equal(t, []float32{2, 3, 4, 5, 6, 7, 8, 9}, dst)
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	n := testing.AllocsPerRun(50, func() {
		if err := out.Eval(t.Context(), CPU, dst); err != nil {
			panic(err)
		}
	})
	require.Equal(t, 0.0, n)
}

func TestGreaterEqual(t *testing.T) {
	a, err := New([]float32{-1, 0, 2}, Shape{3})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 1, 1}, mustEval(t, a.GreaterEqual(Const(float32(0))).Cast[float32]()))
}
