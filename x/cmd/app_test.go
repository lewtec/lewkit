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
	"time"

	"github.com/lewtec/lewkit/x/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppParse(t *testing.T) {
	args, err := Parse[App[None]]("-vv", "--profile-dir", "/tmp/p")
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
			args, err := Parse[App[None]](tc.args...)
			require.NoError(t, err)
			assert.Equal(t, tc.want, args.LogLevel())
		})
	}
}

func TestAppHelpFlag(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		app, err := Parse[App[None]](args...)
		require.NoError(t, err)
		assert.True(t, app.help.Value())
	}
}

func TestAppVersionFlag(t *testing.T) {
	app, err := Parse[App[None]]("--version")
	require.NoError(t, err)
	assert.True(t, app.version.Value())
}

func TestAppUsage(t *testing.T) {
	text, err := Usage[App[None]]("lewkit")
	require.NoError(t, err)
	for _, want := range []string{
		"Usage:",
		"-h, --help",
		"-v, --verbose",
		"log verbosity (default: 0)",
		"--profile-dir",
		"--version",
	} {
		assert.Contains(t, text, want)
	}
	assert.Less(t, strings.Index(text, "Commands:"), strings.Index(text, "Flags:"))
}

func TestAppRunHelp(t *testing.T) {
	app, err := Parse[App[None]]("--help")
	require.NoError(t, err)
	got := captureStdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.True(t, strings.HasPrefix(got, "Usage:"))
	assert.Contains(t, got, "--verbose")
}

func TestAppRunVersion(t *testing.T) {
	cases := [][]string{{"--version"}}
	want := release.Version() + "\n"
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			app, err := Parse[App[None]](args...)
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

	app, err := Parse[App[None]]("-v")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
}

func TestAppRunProfile(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	app, err := Parse[App[None]]("--profile-dir", dir)
	require.NoError(t, err)
	require.NoError(t, app.Run(ctx))

	cpu := filepath.Join(dir, "cpu.prof")
	assert.Eventually(t, func() bool {
		_, err := os.Stat(cpu)
		return err == nil
	}, time.Second, 10*time.Millisecond)
	cancel()
	for _, prof := range pprof.Profiles() {
		path := filepath.Join(dir, prof.Name()+".prof")
		assert.Eventually(t, func() bool {
			stat, err := os.Stat(path)
			return err == nil && stat.Size() > 0
		}, time.Second, 10*time.Millisecond)
	}
}

type extraCmds struct {
	ping *pingCmd
}

type pingCmd struct {
	name StringArg `long:"name"`
	ran  bool
}

func (p *pingCmd) Run(context.Context) error {
	p.ran = true
	return nil
}

func TestAppInnerCommand(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	app, err := Parse[App[extraCmds]]("ping", "--name", "x")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
	require.NotNil(t, app.Args.ping)
	assert.True(t, app.Args.ping.ran)
	assert.Equal(t, "x", app.Args.ping.name.Value())
}

func TestAppInnerHelp(t *testing.T) {
	app, err := Parse[App[extraCmds]]("--help")
	require.NoError(t, err)
	got := captureStdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "ping")
	assert.Contains(t, got, "--verbose")
}

type rootCmd struct {
	name StringArg `long:"name"`
	ran  bool
}

func (r *rootCmd) Run(context.Context) error {
	r.ran = true
	return nil
}

func TestAppRootCommand(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	app, err := Parse[App[rootCmd]]("--name", "x")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
	assert.True(t, app.Args.ran)
	assert.Equal(t, "x", app.Args.name.Value())
}

type describedRoot struct{}

func (describedRoot) Description() string {
	return "inner tool"
}

func TestAppForwardsDescription(t *testing.T) {
	text, err := Usage[App[describedRoot]]("lewkit")
	require.NoError(t, err)
	assert.Contains(t, text, "inner tool")
}
