package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

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
