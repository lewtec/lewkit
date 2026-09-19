package ndarray

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoordEval(t *testing.T) {
	k, err := Compile(Coord(1, 2, 3).Cast(F32))
	require.NoError(t, err)
	got, err := k.Eval()
	require.NoError(t, err)
	require.Equal(t, []float32{0, 1, 2, 0, 1, 2}, got)
}

func TestGeEq(t *testing.T) {
	k, err := Compile(Ge(In(0, st(t, 3)), Const(0)).Cast(F32))
	require.NoError(t, err)
	got, err := k.Eval([]float32{-1, 0, 2})
	require.NoError(t, err)
	require.Equal(t, []float32{0, 1, 1}, got)
}
