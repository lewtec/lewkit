//go:build linux

package exec_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/require"
)

// The helper runs under script so stdin is a terminal. Without the terminal
// check in prepareCancel, the child joins a background process group and
// read stops it with SIGTTIN.
func TestRunReadsFromTerminal(t *testing.T) {
	if os.Getenv("LEWKIT_EXEC_TTY_HELPER") == "1" {
		cmd := execdriver.MustCommand(t.Context(), "sh", "-c", "read x; echo got:$x")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, execdriver.Run(t.Context(), cmd))
		return
	}

	script, err := exec.LookPath("script")
	if err != nil {
		test.Skip(t, "script is not installed")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	argv := strings.Join([]string{os.Args[0], "-test.run", "^TestRunReadsFromTerminal$"}, " ")
	cmd := exec.CommandContext(ctx, script, "-q", "-e", "-c", argv, "/dev/null")
	cmd.Env = append(os.Environ(), "LEWKIT_EXEC_TTY_HELPER=1")
	cmd.Stdin = strings.NewReader("hello\n")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.Contains(t, string(out), "got:hello")
}
