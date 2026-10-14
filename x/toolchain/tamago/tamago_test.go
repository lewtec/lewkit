package tamago

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinkFlags(t *testing.T) {
	want := []string{"-E", "cpuinit", "-T", "0x10010000", "-R", "0x1000"}
	if len(LinkFlags) != len(want) {
		require.FailNowf(t, "", "flags %v", LinkFlags)
	}
	for i := range want {
		if LinkFlags[i] != want[i] {
			require.FailNowf(t, "", "flags %v", LinkFlags)
		}
	}
	if RamStart+0x10000 != 0x10010000 {
		require.FailNowf(t, "", "text address %x", RamStart+0x10000)
	}
}
