package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseGoFlagsAnywhere(t *testing.T) {
	args := normalizeReleaseGoFlags([]string{
		"release", "go", "build", ".",
		"-o", "hello",
		"-C", "examples/app/unikernel",
		"-tags", "extra",
		"-ldflags", "-s -w",
		"-pkgdir", "std",
	})
	app := cmd.ParseOK[cmd.App[root]](t, args...)
	build := app.Args.release.gobuild.build
	require.NotNil(t, build)
	assert.Equal(t, []string{"."}, cmd.Values(build.pkgs))
	assert.Equal(t, "hello", build.output.Value())
	assert.Equal(t, "examples/app/unikernel", build.dir.Value())
	assert.Equal(t, "extra", build.tags.Value())
	assert.Equal(t, "-s -w", build.ldflags.Value())
	assert.Equal(t, "std", build.pkgdir.Value())

	first := normalizeReleaseGoFlags([]string{
		"release", "go", "build",
		"-o", "hello", "-tags", "extra", ".",
	})
	again := cmd.ParseOK[cmd.App[root]](t, first...)
	assert.Equal(t, []string{"."}, cmd.Values(again.Args.release.gobuild.build.pkgs))
	assert.Equal(t, "hello", again.Args.release.gobuild.build.output.Value())
	assert.Equal(t, "extra", again.Args.release.gobuild.build.tags.Value())
}

func TestReleaseGoRunArgs(t *testing.T) {
	args := normalizeReleaseGoFlags([]string{
		"release", "go", "run", ".", "-o", "hello", "--", "-tags",
	})
	app := cmd.ParseOK[cmd.App[root]](t, args...)
	run := app.Args.release.gobuild.run
	require.NotNil(t, run)
	assert.Equal(t, []string{"."}, cmd.Values(run.pkgs))
	assert.Equal(t, "hello", run.output.Value())
	assert.Equal(t, []string{"-tags"}, cmd.Values(run.args))
}
