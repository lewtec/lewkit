package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseBuildOmitsSDKAsSimulator(t *testing.T) {
	app, err := cmd.Parse[cmd.App[root]]("release", "build")
	require.NoError(t, err)
	require.Equal(t, "iphonesimulator", app.Args.release.build.sdk.Value())
	require.False(t, app.Args.release.build.sdk.ArgSet())

	all, err := cmd.Parse[cmd.App[root]]("release", "build-all")
	require.NoError(t, err)
	require.Equal(t, "iphonesimulator", all.Args.release.buildAll.sdk.Value())
	require.False(t, all.Args.release.buildAll.sdk.ArgSet())
}

func TestReleaseRunSelectsSimulator(t *testing.T) {
	var run runCmd
	require.NoError(t, run.goos.Parse("ios"))
	require.NoError(t, run.app.Count(1))
	require.NoError(t, run.prepareIOSRun())
	require.Equal(t, "iphonesimulator", run.sdk.Value())

	run.sdk = cmd.StringArg{}
	require.NoError(t, run.sdk.Parse("iphoneos"))
	run.sdk.MarkSet()
	err := run.prepareIOSRun()
	require.ErrorIs(t, err, errIOSRunNeedsSimulator)

	var desktop runCmd
	require.NoError(t, desktop.goos.Parse("darwin"))
	require.NoError(t, desktop.app.Count(1))
	require.NoError(t, desktop.prepareIOSRun())
	require.Empty(t, desktop.sdk.Value())
}

func TestForegroundCommandUsesProcessStreams(t *testing.T) {
	cmd := foregroundCommand(t.Context(), "true")
	assert.Same(t, os.Stdin, cmd.Stdin)
	assert.Same(t, os.Stdout, cmd.Stdout)
	assert.Same(t, os.Stderr, cmd.Stderr)
}

func TestRunForegroundRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := runForeground(ctx, "true")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunForegroundLeavesCancelToChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "sleep")
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	require.NoError(t, runForeground(ctx, "sleep", "0.2"))
}

func TestAndroidComponent(t *testing.T) {
	require.Equal(t, "br.tec.lew.drivers/lewkit.MainActivity", androidComponent("br.tec.lew.drivers"))
}

func TestAdbBinUsesPath(t *testing.T) {
	dir := t.TempDir()
	name := "adb"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "sdk"))
	t.Setenv("ANDROID_SDK_ROOT", "")
	got, err := adbBin(t.Context())
	require.NoError(t, err)
	require.Equal(t, bin, got)
}

func TestLaunchWindowsAppNeedsWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "this machine can start the exe")
	}
	err := launchApp(t.Context(), "windows", "Demo.exe", "br.tec.lew.demo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "windows")
}

func TestRunBuiltRejectsOtherPlatform(t *testing.T) {
	err := runBuilt(t.Context(), builtProgram{goos: "js", goarch: "wasm", archive: "unused"})
	require.ErrorIs(t, err, errOtherPlatform)
}

func scriptArchive(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	payload := []byte(body)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "prog",
		Mode: 0o755,
		Size: int64(len(payload)),
	}))
	_, err = tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
	return path
}

func TestRunBuiltPassesArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "shell script")
	}
	archive := scriptArchive(t, "#!/bin/sh\ntest \"$1\" = smoke\n")
	err := runBuilt(t.Context(), builtProgram{
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		archive: archive,
		args:    []string{"smoke"},
	})
	require.NoError(t, err)
	err = runBuilt(t.Context(), builtProgram{
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		archive: archive,
	})
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}
