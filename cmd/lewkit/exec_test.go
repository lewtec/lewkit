package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecRegistersConda(t *testing.T) {
	backend, err := tool.Get("conda")
	require.NoError(t, err)
	require.Equal(t, "conda", backend.Name())
}

func TestExecParsesEnvTargetAndCommand(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.env")
	second := filepath.Join(dir, "b.env")
	require.NoError(t, os.WriteFile(first, []byte("A=1\n"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("B=2\n"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "exec", "-e", first, "-e", second, "-t", "conda:clang", "-t", "mise:go", "--", "go", "--help")
	require.Len(t, app.Args.exec.env, 2)
	assert.Equal(t, "A=1\n", string(app.Args.exec.env[0].Value()))
	assert.Equal(t, "B=2\n", string(app.Args.exec.env[1].Value()))
	assert.Equal(t, []string{"conda:clang", "mise:go"}, cmd.Values(app.Args.exec.tools))
	assert.Equal(t, []string{"go", "--help"}, cmd.Values(app.Args.exec.args))
}

func TestExecDashKeepsHelpArgument(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "exec", "--", "go", "--help")
	assert.Equal(t, []string{"go", "--help"}, cmd.Values(app.Args.exec.args))
}

func TestExecStartsCommandAfterProgress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	script := filepath.Join(dir, "mark")
	body := "#!/bin/sh\n: > " + strconv.Quote(marker) + "\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))

	app := cmd.ParseOK[cmd.App[root]](t, "exec", "--", script)
	require.NoError(t, entry.Run(t.Context(), func(ctx context.Context) error {
		err := app.Args.exec.Run(ctx)
		_, statErr := os.Stat(marker)
		require.ErrorIs(t, statErr, os.ErrNotExist)
		return err
	}))
	_, err := os.Stat(marker)
	require.NoError(t, err)
}

func TestExecRunsImmediatelyWithoutSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	script := filepath.Join(dir, "mark")
	body := "#!/bin/sh\n: > " + strconv.Quote(marker) + "\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))

	app := cmd.ParseOK[cmd.App[root]](t, "exec", "--", script)
	require.NoError(t, app.Args.exec.Run(t.Context()))
	_, err := os.Stat(marker)
	require.NoError(t, err)
}
