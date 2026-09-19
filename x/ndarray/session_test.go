package ndarray

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func vmSizeKB(t *testing.T) int64 {
	t.Helper()
	b, err := os.ReadFile("/proc/self/status")
	require.NoError(t, err)
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "VmSize:") {
			continue
		}
		f := strings.Fields(line)
		require.GreaterOrEqual(t, len(f), 2)
		n, err := strconv.ParseInt(f[1], 10, 64)
		require.NoError(t, err)
		return n
	}
	t.Fatal("no VmSize")
	return 0
}

func compileAdd1(t *testing.T, n int) (*Kernel, *Tensor) {
	t.Helper()
	a, err := New(make([]float32, n), Shape{n})
	require.NoError(t, err)
	k, err := compile(a.Add(Const(1)).node)
	require.NoError(t, err)
	return k, a
}

func TestSessionRunLoop(t *testing.T) {
	k, a := compileAdd1(t, 8)
	test.CloseOnCleanup(t, k)
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	s, err := k.Attach(t.Context(), d)
	require.NoError(t, err)
	test.CloseOnCleanup(t, s)
	copy(a.Buffer(), []float32{1, 2, 3, 4, 5, 6, 7, 8})
	dst := make([]float32, 8)
	for range 20 {
		require.NoError(t, s.Run(t.Context(), dst))
	}
	require.Equal(t, []float32{2, 3, 4, 5, 6, 7, 8, 9}, dst)
}

func TestSessionVirtStable(t *testing.T) {
	k, _ := compileAdd1(t, 4096)
	test.CloseOnCleanup(t, k)
	dst := make([]float32, 4096)

	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	s, err := k.Attach(t.Context(), d)
	require.NoError(t, err)
	test.CloseOnCleanup(t, s)
	require.NoError(t, s.Run(t.Context(), dst))
	v0 := vmSizeKB(t)
	for range 80 {
		require.NoError(t, s.Run(t.Context(), dst))
	}
	v1 := vmSizeKB(t)
	grew := v1 - v0
	t.Logf("VmSize %d -> %d kB (%+d) over 80 GPU runs", v0, v1, grew)
	require.Less(t, grew, int64(64*1024), "virtual size grew %d kB", grew)
}

func TestEvalIntoVirtStable(t *testing.T) {
	k, _ := compileAdd1(t, 4096)
	dst := make([]float32, 4096)
	require.NoError(t, k.EvalInto(dst))
	v0 := vmSizeKB(t)
	for range 80 {
		require.NoError(t, k.EvalInto(dst))
	}
	v1 := vmSizeKB(t)
	grew := v1 - v0
	t.Logf("VmSize %d -> %d kB (%+d) over 80 CPU evals", v0, v1, grew)
	require.Less(t, grew, int64(64*1024), "virtual size grew %d kB", grew)
}

func TestSessionCPU(t *testing.T) {
	a, err := New([]float32{1, 2, 3, 4}, Shape{4})
	require.NoError(t, err)
	k, err := compile(a.Mul(Const(2)).node)
	require.NoError(t, err)
	s, err := k.Attach(t.Context(), nil)
	require.NoError(t, err)
	dst := make([]float32, 4)
	require.NoError(t, s.Run(t.Context(), dst))
	require.Equal(t, []float32{2, 4, 6, 8}, dst)
}
