package nodescript

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	stub := filepath.Join(dir, "stub")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\nprintf '%s\\n' \"$PATH\" >> " + shellQuote(logPath) + "\n"
	require.NoError(t, os.WriteFile(stub, []byte(script), 0o755))

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

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
