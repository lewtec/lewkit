package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecParsesEnvTargetAndCommand(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.env")
	second := filepath.Join(dir, "b.env")
	require.NoError(t, os.WriteFile(first, []byte("A=1\n"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("B=2\n"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "exec", "-e", first, "-e", second, "-t", "conda:foo", "--", "echo", "hi")
	require.Len(t, app.Args.exec.env, 2)
	assert.Equal(t, "A=1\n", string(app.Args.exec.env[0].Value()))
	assert.Equal(t, "B=2\n", string(app.Args.exec.env[1].Value()))
	assert.Equal(t, "conda:foo", app.Args.exec.target.Value())
	assert.Equal(t, []string{"echo", "hi"}, cmd.Values(app.Args.exec.args))
}

func TestExecDashKeepsHelpArgument(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "exec", "--", "go", "--help")
	assert.Equal(t, []string{"go", "--help"}, cmd.Values(app.Args.exec.args))
}
