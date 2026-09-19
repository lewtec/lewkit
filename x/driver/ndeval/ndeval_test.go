package ndeval

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	evaluator, err := ndarray.Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	x, err := ndarray.Ones(ndarray.Shape{4})
	require.NoError(t, err)
	dst := make([]float32, 4)
	require.NoError(t, x.Eval(t.Context(), evaluator, dst))
	require.Equal(t, []float32{1, 1, 1, 1}, dst)
}
