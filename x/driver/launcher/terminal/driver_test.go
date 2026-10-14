package terminal

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisabledWhenStdioIsNotATTY(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("TERMUX_VERSION", "")
	read, write, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		read.Close()
		write.Close()
	})
	oldOut := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = oldOut })
	if err := (base{}).CheckCompatibility(t.Context()); err == nil {
		require.FailNow(t, fmt.Sprint("terminal launcher is compatible when stdout is a pipe"))
	}
}
