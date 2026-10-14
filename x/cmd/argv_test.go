package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNameAndArgs(t *testing.T) {
	if got, want := Name(), filepath.Base(os.Args[0]); got != want {
		require.FailNowf(t, "", "Name() = %q, want %q", got, want)
	}
	got := Args()
	want := os.Args[1:]
	if len(got) != len(want) {
		require.FailNowf(t, "", "Args() len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			require.FailNowf(t, "", "Args()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
