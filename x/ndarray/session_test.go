package ndarray

import (
	"os"
	"strconv"
	"strings"
	"testing"

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

func TestEvalIntoVirtStable(t *testing.T) {
	a, err := New(make([]float32, 256), Shape{256})
	require.NoError(t, err)
	out := a.Add(Const(1))
	dst := make([]float32, 256)
	require.NoError(t, out.Eval(t.Context(), CPU, dst))
	v0 := vmSizeKB(t)
	for range 80 {
		require.NoError(t, out.Eval(t.Context(), CPU, dst))
	}
	v1 := vmSizeKB(t)
	grew := v1 - v0
	t.Logf("VmSize %d -> %d kB (%+d) over 80 CPU evals", v0, v1, grew)
	require.Less(t, grew, int64(64*1024), "virtual size grew %d kB", grew)
}
