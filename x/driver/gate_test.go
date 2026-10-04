package driver_test

import (
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestAppMode(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("TERMUX_VERSION", "")
	require.False(t, driver.AppMode())

	t.Setenv("ELETROCROMO_NO_UI", "1")
	require.True(t, driver.AppMode())

	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("LEWKIT_NO_UI", "true")
	require.True(t, driver.AppMode())

	t.Setenv("TERMUX_VERSION", "0.118")
	require.False(t, driver.AppMode())
}

func TestMemoryGate(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "")
	require.ErrorIs(t, driver.MemoryGate(), driver.ErrIncompatible)
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	require.NoError(t, driver.MemoryGate())
}

func TestTerminalGate(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("TERMUX_VERSION", "")

	read, write, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		read.Close()
		write.Close()
	})
	oldIn, oldOut := os.Stdin, os.Stdout
	t.Cleanup(func() { os.Stdin, os.Stdout = oldIn, oldOut })

	os.Stdin, os.Stdout = read, write
	require.ErrorIs(t, driver.TerminalGate(), driver.ErrIncompatible)

	os.Stdout = oldOut
	require.ErrorIs(t, driver.TerminalGate(), driver.ErrIncompatible)

	os.Stdin, os.Stdout = oldIn, write
	require.ErrorIs(t, driver.TerminalGate(), driver.ErrIncompatible)

	os.Stdin, os.Stdout = oldIn, oldOut
	t.Setenv("ELETROCROMO_NO_UI", "1")
	require.ErrorIs(t, driver.TerminalGate(), driver.ErrIncompatible)
}
