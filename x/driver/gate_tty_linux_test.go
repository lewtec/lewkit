//go:build linux

package driver_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func TestTerminalGateOnTTY(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("TERMUX_VERSION", "")

	master, slave := openPTY(t)
	t.Cleanup(func() {
		master.Close()
		slave.Close()
	})
	require.True(t, term.IsTerminal(int(slave.Fd())))

	oldIn, oldOut := os.Stdin, os.Stdout
	t.Cleanup(func() { os.Stdin, os.Stdout = oldIn, oldOut })
	os.Stdin, os.Stdout = slave, slave
	require.NoError(t, driver.TerminalGate())

	t.Setenv("ELETROCROMO_NO_UI", "1")
	require.ErrorIs(t, driver.TerminalGate(), driver.ErrIncompatible)
}

func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	require.NoError(t, err)
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	return master, slave
}
