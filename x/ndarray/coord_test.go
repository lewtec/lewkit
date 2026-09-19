package ndarray

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoordEval(t *testing.T) {
	k, err := Compile(Coord(1, Shape{2, 3}).Cast(F32))
	require.NoError(t, err)
	got, err := k.Eval()
	require.NoError(t, err)
	require.Equal(t, []float32{0, 1, 2, 0, 1, 2}, got)
}

func TestEvalInto(t *testing.T) {
	k, err := Compile(In(0, mustTracker(t, Shape{8})).Add(Const(1)))
	require.NoError(t, err)
	src := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	dst := make([]float32, 8)
	in := [][]float32{src}
	require.NoError(t, k.EvalInto(dst, in))
	require.Equal(t, []float32{2, 3, 4, 5, 6, 7, 8, 9}, dst)
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	n := testing.AllocsPerRun(50, func() {
		if err := k.EvalInto(dst, in); err != nil {
			panic(err)
		}
	})
	require.Equal(t, 0.0, n)
}

func TestGreaterEqual(t *testing.T) {
	k, err := Compile(GreaterEqual(In(0, mustTracker(t, Shape{3})), Const(0)).Cast(F32))
	require.NoError(t, err)
	got, err := k.Eval([]float32{-1, 0, 2})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 1, 1}, got)
}
