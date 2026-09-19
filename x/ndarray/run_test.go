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
	a, err := New([]float32{-1, 2, -3, 4}, Shape{4})
	require.NoError(t, err)
	b, err := New([]float32{2, -1, 5, -1}, Shape{4})
	require.NoError(t, err)
	expr := a.Add(b).Max(Const(0))
	want := append([]float32(nil), mustEval(t, expr)...)
	require.NoError(t, expr.Exec(t.Context(), d))
	got, err := expr.Data()
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExecPermute(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	a, err := New([]float32{1, 2, 3, 4, 5, 6}, Shape{3, 2})
	require.NoError(t, err)
	raw, err := New([]float32{10, 20, 30, 40, 50, 60}, Shape{2, 3})
	require.NoError(t, err)
	b, err := raw.Permute(1, 0)
	require.NoError(t, err)
	expr := a.Add(b)
	want := append([]float32(nil), mustEval(t, expr)...)
	require.NoError(t, expr.Exec(t.Context(), d))
	got, err := expr.Data()
	require.NoError(t, err)
	require.Equal(t, want, got)
}
