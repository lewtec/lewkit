package ndarray

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestSessionRunLoop(t *testing.T) {
	k, err := Compile(In(0, st(t, 8)).Add(Const(1)))
	require.NoError(t, err)
	test.CloseOnCleanup(t, k)
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	s, err := k.Attach(t.Context(), d)
	require.NoError(t, err)
	test.CloseOnCleanup(t, s)
	src := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	dst := make([]float32, 8)
	in := [][]float32{src}
	for range 20 {
		require.NoError(t, s.Run(t.Context(), dst, in))
	}
	require.Equal(t, []float32{2, 3, 4, 5, 6, 7, 8, 9}, dst)
}

func TestSessionCPU(t *testing.T) {
	k, err := Compile(In(0, st(t, 4)).Mul(Const(2)))
	require.NoError(t, err)
	s, err := k.Attach(t.Context(), nil)
	require.NoError(t, err)
	dst := make([]float32, 4)
	require.NoError(t, s.Run(t.Context(), dst, [][]float32{{1, 2, 3, 4}}))
	require.Equal(t, []float32{2, 4, 6, 8}, dst)
}
