package ndarray

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestExecAdd(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	k, err := Compile(In(0, st(t, 4)).Add(In(1, st(t, 4))).Max(Const(0)))
	require.NoError(t, err)
	test.CloseOnCleanup(t, k)
	a := []float32{-1, 2, -3, 4}
	b := []float32{2, -1, 5, -1}
	want, err := k.Eval(a, b)
	require.NoError(t, err)
	got, err := k.Exec(t.Context(), d, a, b)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExecPermute(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	perm, err := st(t, 2, 3).Permute(1, 0)
	require.NoError(t, err)
	k, err := Compile(In(0, st(t, 3, 2)).Add(In(1, perm)))
	require.NoError(t, err)
	test.CloseOnCleanup(t, k)
	a := []float32{1, 2, 3, 4, 5, 6}
	b := []float32{10, 20, 30, 40, 50, 60}
	want, err := k.Eval(a, b)
	require.NoError(t, err)
	got, err := k.Exec(t.Context(), d, a, b)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
