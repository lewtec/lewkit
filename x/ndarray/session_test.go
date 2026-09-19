package ndarray

import (
	"os"
	"testing"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/require"
)

func TestEvalIntoVirtStable(t *testing.T) {
	a, err := New(make([]float32, 256), Shape{256})
	require.NoError(t, err)
	out := a.Add(Const(float32(1)))
	dst := make([]float32, 256)
	require.NoError(t, out.Eval(t.Context(), CPU, dst))
	proc, err := process.NewProcess(int32(os.Getpid()))
	require.NoError(t, err)
	before, err := proc.MemoryInfo()
	require.NoError(t, err)
	for range 80 {
		require.NoError(t, out.Eval(t.Context(), CPU, dst))
	}
	after, err := proc.MemoryInfo()
	require.NoError(t, err)
	grew := int64(after.VMS) - int64(before.VMS)
	t.Logf("VMS %d -> %d (%+d) RSS %d -> %d over 80 CPU evals", before.VMS, after.VMS, grew, before.RSS, after.RSS)
	require.Less(t, grew, int64(64<<20), "virtual size grew %d bytes", grew)
}
