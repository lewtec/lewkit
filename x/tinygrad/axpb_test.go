package tinygrad

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

// TestAxPlusB is y = A x + b: views pick A columns and broadcast x, then
// MUL/ADD fuse into one kernel.
func TestAxPlusB(t *testing.T) {
	const m, k = 2, 3
	A := []float32{
		1, 2, 3,
		4, 5, 6,
	}
	x := []float32{10, 20, 30}
	b := []float32{100, 200}
	want := []float32{240, 520} // [1 2 3; 4 5 6] [10 20 30] + [100 200]

	expr := axPlusB(t, m, k)
	kn, err := Compile(expr)
	require.NoError(t, err)
	require.Equal(t, []int{m}, kn.Shape())
	require.Equal(t, []int{0, 1, 2}, kn.Slots())
	require.Equal(t, 1, strings.Count(kn.GLSL(), "void main()"))

	got, err := kn.Eval(A, x, b)
	require.NoError(t, err)
	require.Equal(t, want, got)

	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Log(err)
		return
	}
	test.CloseOnCleanup(t, d)
	gpu, err := kn.Exec(t.Context(), d, A, x, b)
	require.NoError(t, err)
	require.Equal(t, want, gpu)
}

func axPlusB(t *testing.T, m, k int) *Node {
	t.Helper()
	A := st(t, m, k)
	x := st(t, k)
	bias := st(t, m)
	var acc *Node
	for j := range k {
		col, err := A.Shrink([][2]int{{0, m}, {j, j + 1}})
		require.NoError(t, err)
		col, err = col.Reshape(m)
		require.NoError(t, err)
		xj, err := x.Shrink([][2]int{{j, j + 1}})
		require.NoError(t, err)
		xj, err = xj.Expand(m)
		require.NoError(t, err)
		term := In(0, col).Mul(In(1, xj))
		if acc == nil {
			acc = term
		} else {
			acc = acc.Add(term)
		}
	}
	return acc.Add(In(2, bias))
}
