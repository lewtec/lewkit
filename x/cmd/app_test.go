package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppParse(t *testing.T) {
	args, err := Parse[App]("-vv", "--profile-dir", "/tmp/p")
	require.NoError(t, err)
	assert.Equal(t, 2, args.verbose.Value())
	assert.Equal(t, "/tmp/p", args.profileDir.Value())
	assert.Equal(t, slog.LevelDebug-4, args.LogLevel())
}

func TestAppLogLevel(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want slog.Level
	}{
		{name: "default", want: slog.LevelInfo},
		{name: "one", args: []string{"-v"}, want: slog.LevelDebug},
		{name: "count value", args: []string{"--verbose", "2"}, want: slog.LevelDebug - 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := Parse[App](tc.args...)
			require.NoError(t, err)
			assert.Equal(t, tc.want, args.LogLevel())
		})
	}
}

func TestAppHelpFlag(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		app, err := Parse[App](args...)
		require.NoError(t, err)
		assert.True(t, app.help.Value())
	}
}

func TestAppVersionFlag(t *testing.T) {
	app, err := Parse[App]("--version")
	require.NoError(t, err)
	assert.True(t, app.version.Value())
	assert.Nil(t, app.versionCmd)
}

func TestAppVersionCommand(t *testing.T) {
	app, err := Parse[App]("version")
	require.NoError(t, err)
	assert.False(t, app.version.Value())
	require.NotNil(t, app.versionCmd)
}

func TestAppUsage(t *testing.T) {
	text, err := Usage[App]("lewkit")
	require.NoError(t, err)
	for _, want := range []string{
		"Usage:",
		"lewkit <command> [args]",
		"-h, --help",
		"-v, --verbose",
		"--profile-dir",
		"--version",
		"Commands:",
		"version",
	} {
		assert.Contains(t, text, want)
	}
}

func TestAppRunHelp(t *testing.T) {
	app, err := Parse[App]("--help")
	require.NoError(t, err)
	got := captureStdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.True(t, strings.HasPrefix(got, "Usage:"))
	assert.Contains(t, got, "--verbose")
}

func TestAppRunVersion(t *testing.T) {
	cases := [][]string{{"--version"}, {"version"}}
	want := release.Version() + "\n"
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			app, err := Parse[App](args...)
			require.NoError(t, err)
			got := captureStdout(t, func() {
				require.NoError(t, app.Run(t.Context()))
			})
			assert.Equal(t, want, got)
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })
	fn()
	require.NoError(t, w.Close())
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(got)
}

func TestAppRunNoProfile(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	app, err := Parse[App]("-v")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
}

func TestAppRunProfile(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	app, err := Parse[App]("--profile-dir", dir)
	require.NoError(t, err)
	require.NoError(t, app.Run(ctx))

	for _, prof := range pprof.Profiles() {
		stat, err := os.Stat(filepath.Join(dir, prof.Name()+".prof"))
		require.NoError(t, err)
		assert.NotZero(t, stat.Size())
	}
}
