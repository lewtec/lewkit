package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgramGoOnly(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--go-only")
	fn, err := app.Args.release.run.program(t.Context(), []string{"a.tar.gz"})
	require.NoError(t, err)
	assert.Nil(t, fn)
}

func TestProgramEmptyPaths(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run")
	fn, err := app.Args.release.run.program(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, fn)
}

func TestProgramHostRejectsArgs(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--app", "--config", "eletrocromo.json", "--", "nope")
	_, err := app.Args.release.run.program(t.Context(), []string{"app.apk"})
	require.ErrorIs(t, err, errHostArgs)
}

func TestConfigDefaultsToEletrocromoJSON(t *testing.T) {
	run := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--app")
	assert.Equal(t, "./eletrocromo.json", run.Args.release.run.config.Value())
	build := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	assert.Equal(t, "./eletrocromo.json", build.Args.release.build.config.Value())
}

func TestP12FlagReadsSecretFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "publisher.p12")
	passPath := filepath.Join(dir, "password.txt")
	require.NoError(t, os.WriteFile(keyPath, []byte{0x30, 0x82}, 0o600))
	require.NoError(t, os.WriteFile(passPath, []byte("hunter2\n"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--p12", keyPath, "--p12-password", passPath)
	assert.Equal(t, []byte{0x30, 0x82}, app.Args.release.build.p12.Value())
	assert.Equal(t, "hunter2", app.Args.release.build.p12Password.Text())

	_, err := app.Args.release.build.identity()
	require.ErrorIs(t, err, sign.ErrPKCS12)
}

func TestP12EnvReadsSecretFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(keyPath, []byte("pkcs"), 0o600))
	t.Setenv("LEWKIT_SIGN_P12", keyPath)
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")

	app := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	assert.Equal(t, []byte("pkcs"), app.Args.release.build.p12.Value())
	assert.Empty(t, app.Args.release.build.p12Password.Text())
}

func TestIdentityUnset(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	id, err := app.Args.release.build.identity()
	require.NoError(t, err)
	assert.Nil(t, id)
}

func TestAppUsesDefaultConfig(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--app", "--go-only")
	_, err := app.Args.release.build.produce(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestProgramStartsAfterProgress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	archive := scriptArchive(t, "#!/bin/sh\ntest \"$1\" = smoke\n")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--", "smoke")
	fn, err := app.Args.release.run.program(t.Context(), []string{archive})
	require.NoError(t, err)
	require.NotNil(t, fn)
	require.NoError(t, entry.Run(t.Context(), func(context.Context) error {
		entry.After(fn)
		return nil
	}))
}

func TestRunNestedReturnsBuildError(t *testing.T) {
	session, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--config", filepath.Join(t.TempDir(), "missing.json"))
	var runErr error
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		runErr = app.Args.release.run.Run(ctx)
		return nil
	})
	require.Error(t, runErr)
	require.Error(t, err)
}
