package ndarray

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestEvalIntoVirtStable(t *testing.T) {
	a, err := New(make([]float32, 256), Shape{256})
	require.NoError(t, err)
	out := a.Add(Const(1))
	dst := make([]float32, 256)
	require.NoError(t, out.Eval(t.Context(), CPU, dst))
	v0 := test.VirtSize(t)
	for range 80 {
		require.NoError(t, out.Eval(t.Context(), CPU, dst))
	}
	v1 := test.VirtSize(t)
	grew := v1 - v0
	t.Logf("virt %d -> %d (%+d) over 80 CPU evals", v0, v1, grew)
	require.Less(t, grew, int64(64<<20), "virtual size grew %d bytes", grew)
}
