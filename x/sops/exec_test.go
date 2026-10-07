package sops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/lewtec/lewkit/x/driver/exec/native"
)

func TestEnvParseMerge(t *testing.T) {
	first, err := ParseEnv([]byte("# note\nA=1\nB=2\nP=\"hunter2\"\nMSG=a\\nb\n"))
	require.NoError(t, err)
	second, err := ParseEnv([]byte("B=3\nC=4\n"))
	require.NoError(t, err)
	merged := first.Merge(second)
	require.Equal(t, []string{"A=1", "B=3", `P="hunter2"`, "MSG=a\nb", "C=4"}, merged.Strings())
	got, ok := merged.Get("MSG")
	require.True(t, ok)
	require.Equal(t, "a\nb", got)
	_, ok = merged.Get("missing")
	require.False(t, ok)
}

func TestEnvRejectsBadLine(t *testing.T) {
	_, err := ParseEnv([]byte("not-an-assignment\n"))
	require.Error(t, err)
}

func TestCommandAppliesEnv(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	env, err := ParseEnv([]byte("LEWKIT_EXEC_SENTINEL=from-file\n"))
	require.NoError(t, err)
	err = (Command{
		Env:  env,
		Args: []string{"sh", "-c", `printf %s "$LEWKIT_EXEC_SENTINEL" > ` + out},
	}).Run(t.Context())
	require.NoError(t, err)
	got, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "from-file", string(got))
}

func TestCommandExitStatus(t *testing.T) {
	err := (Command{Args: []string{"sh", "-c", "exit 3"}}).Run(t.Context())
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 3, exitErr.ExitCode())
}

func TestCommandEmpty(t *testing.T) {
	err := (Command{}).Run(t.Context())
	require.Error(t, err)
}

func TestLoadEnvDotenv(t *testing.T) {
	useAgeKey(t)
	env, err := LoadEnv("testdata/env.env")
	require.NoError(t, err)
	password, ok := env.Get("password")
	require.True(t, ok)
	require.Equal(t, "hunter2", password)
	note, ok := env.Get("note_unencrypted")
	require.True(t, ok)
	require.Equal(t, "visible", note)
	token, ok := env.Get("token")
	require.True(t, ok)
	require.Equal(t, "line1\nline2", token)
	_, ok = env.Get("sops_mac")
	require.False(t, ok)
}
