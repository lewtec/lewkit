package nodescript

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestForLockfile(t *testing.T) {
	dir := t.TempDir()
	_, err := For(dir)
	require.ErrorIs(t, err, ErrLockfile)
	require.ErrorIs(t, err, errs.ErrNotFound)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("# yarn\n"), 0o644))
	_, err = For(dir)
	require.ErrorIs(t, err, ErrLockfile)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}\n"), 0o644))
	got, err := For(dir)
	require.NoError(t, err)
	require.Equal(t, Manager{Command: "npm", Spec: "nodejs", Binary: "npm"}, got)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9\n"), 0o644))
	got, err = For(dir)
	require.NoError(t, err)
	require.Equal(t, "pnpm", got.Command)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "bun.lock"), []byte("{}\n"), 0o644))
	got, err = For(dir)
	require.NoError(t, err)
	require.Equal(t, "bun", got.Command)

	lockb := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(lockb, "bun.lockb"), []byte("bin"), 0o644))
	got, err = For(lockb)
	require.NoError(t, err)
	require.Equal(t, "bun", got.Command)
}

func TestRunInstallsThenScript(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bun.lock"), []byte("{}\n"), 0o644))
	logPath := filepath.Join(dir, "calls.txt")
	t.Setenv("CALLS", logPath)
	stub := writeStub(t, dir)

	previous := resolveBinary
	resolveBinary = func(_ context.Context, manager Manager) (string, error) {
		require.Equal(t, "bun", manager.Command)
		return stub, nil
	}
	t.Cleanup(func() { resolveBinary = previous })

	require.NoError(t, Run(t.Context(), dir, "build"))
	body, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	require.Equal(t, "install", lines[0])
	require.True(t, strings.HasSuffix(lines[1], filepath.Join("node_modules", ".bin")))
	require.Equal(t, "run build", lines[2])
	require.True(t, strings.HasSuffix(lines[3], filepath.Join("node_modules", ".bin")))
}

func TestRunRejectsEmptyScript(t *testing.T) {
	err := Run(t.Context(), t.TempDir(), " ")
	require.ErrorIs(t, err, ErrScript)
}

func writeStub(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "stubsrc")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.go"), []byte(stubMain), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"), []byte("module stub\n\ngo 1.27.0\n"), 0o644))
	out := filepath.Join(dir, "stub")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := execdriver.MustCommand(t.Context(), "go", "build", "-o", out, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	require.NoError(t, execdriver.Run(t.Context(), cmd))
	return out
}

const stubMain = `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	file, err := os.OpenFile(os.Getenv("CALLS"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()
	fmt.Fprintln(file, strings.Join(os.Args[1:], " "))
	fmt.Fprintln(file, os.Getenv("PATH"))
}
`
