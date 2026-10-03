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
	sample := func() (uint64, uint64) {
		info, err := proc.MemoryInfo()
		require.NoError(t, err)
		return info.VMS, info.RSS
	}
	eval := func() {
		for range 80 {
			require.NoError(t, out.Eval(t.Context(), CPU, dst))
		}
	}
	v0, r0 := sample()
	eval()
	v1, r1 := sample()
	eval()
	v2, r2 := sample()
	first := int64(v1) - int64(v0)
	second := int64(v2) - int64(v1)
	t.Logf("VMS %d -> %d -> %d (%+d, %+d) RSS %d -> %d -> %d", v0, v1, v2, first, second, r0, r1, r2)
	// The runtime reserves virtual address space in 64 MiB arenas. Crossing a
	// boundary maps one arena. A leak maps another on the next run.
	const arena = int64(64 << 20)
	count := func(grew int64) (int, int64) {
		if grew <= 0 {
			return 0, 0
		}
		return int(grew / arena), grew % arena
	}
	arenas1, rest1 := count(first)
	arenas2, rest2 := count(second)
	require.LessOrEqual(t, arenas1+arenas2, 1, "virtual size grew %+d then %+d", first, second)
	require.Less(t, rest1+rest2, int64(1<<20), "virtual size grew %+d then %+d", first, second)
}
