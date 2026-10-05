package metal

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	ffimetal "github.com/lewtec/lewkit/x/ffi/native/metal"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestFactoryIdentity(t *testing.T) {
	require.Equal(t, "ndeval_metal", factory{}.ID())
	require.Equal(t, 80, factory{}.Weight())
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestEvalAdd(t *testing.T) {
	if err := ffimetal.Available(); err != nil {
		t.Skip(err)
	}
	evaluator, err := factory{}.New(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	a, err := ndarray.New([]float32{-1, 2, -3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{2, -1, 5, -1}, ndarray.Shape{4})
	require.NoError(t, err)
	expr := a.Add(b).Max(ndarray.Const(float32(0)))
	want := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), ndarray.CPU, want))
	got := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), evaluator, got))
	require.Equal(t, want, got)
}
