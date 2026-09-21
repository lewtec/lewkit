package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppParse(t *testing.T) {
	args := ParseOK[App[None]](t, "-vv", "--pprof", "/tmp/p")
	assert.Equal(t, 2, args.verbose.Value())
	assert.Equal(t, "/tmp/p", args.pprof.Value())
	assert.Equal(t, slog.LevelDebug-4, args.LogLevel())
}

func TestPprofClassifies(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		directory string
		address   string
	}{
		{name: "absolute dir", in: "/tmp/p", directory: "/tmp/p"},
		{name: "relative dir", in: "./out", directory: "./out"},
		{name: "name", in: "profiles", directory: "profiles"},
		{name: "all interfaces", in: ":6060", address: ":6060"},
		{name: "localhost", in: "localhost:6060", address: "localhost:6060"},
		{name: "ipv4", in: "127.0.0.1:6060", address: "127.0.0.1:6060"},
		{name: "bare port", in: "8080", address: ":8080"},
		{name: "ipv6", in: "[::1]:443", address: "[::1]:443"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newProfile(tc.in)
			assert.Equal(t, tc.directory, p.Directory())
			assert.Equal(t, tc.address, p.Address())
		})
	}
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
			args := ParseOK[App[None]](t, tc.args...)
			assert.Equal(t, tc.want, args.LogLevel())
		})
	}
}

func TestAppHelpFlag(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		app := ParseOK[App[None]](t, args...)
		assert.True(t, app.help.Value())
	}
}

func TestAppVersionFlag(t *testing.T) {
	app := ParseOK[App[None]](t, "--version")
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
		"--pprof",
		"--version",
	} {
		assert.Contains(t, text, want)
	}
	assert.Less(t, strings.Index(text, "Commands:"), strings.Index(text, "Flags:"))
}

func TestAppRunHelp(t *testing.T) {
	app := ParseOK[App[None]](t, "--help")
	got := test.Stdout(t, func() {
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
			app := ParseOK[App[None]](t, args...)
			got := test.Stdout(t, func() {
				require.NoError(t, app.Run(t.Context()))
			})
			assert.Equal(t, want, got)
		})
	}
}

func TestAppRunNoPprof(t *testing.T) {
	test.RestoreSlog(t)

	app := ParseOK[App[None]](t, "-v")
	require.NoError(t, app.Run(t.Context()))
}

func TestAppRunCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	app := ParseOK[App[None]](t)
	assert.ErrorIs(t, app.Run(ctx), context.Canceled)
}

func TestAppRunPprof(t *testing.T) {
	test.RestoreSlog(t)

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	app := ParseOK[App[None]](t, "--pprof", dir)
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

func TestMissingCommand(t *testing.T) {
	test.RestoreSlog(t)

	app := ParseOK[App[extraCmds]](t)
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "Usage:")
	assert.Contains(t, got, "ping")
}

func TestAppInnerCommand(t *testing.T) {
	test.RestoreSlog(t)

	app := ParseOK[App[extraCmds]](t, "ping", "--name", "x")
	require.NoError(t, app.Run(t.Context()))
	require.NotNil(t, app.Args.ping)
	assert.True(t, app.Args.ping.ran)
	assert.Equal(t, "x", app.Args.ping.name.Value())
}

func TestAppVerboseAfterCommand(t *testing.T) {
	app := ParseOK[App[extraCmds]](t, "ping", "-vv", "--name", "x")
	assert.Equal(t, 2, app.verbose.Value())
	assert.Equal(t, slog.LevelDebug-4, app.LogLevel())
	require.NotNil(t, app.Args.ping)
	assert.Equal(t, "x", app.Args.ping.name.Value())
}

func TestAppHelpAfterCommand(t *testing.T) {
	app := ParseOK[App[extraCmds]](t, "ping", "--help")
	assert.True(t, app.help.Value())
	require.NotNil(t, app.Args.ping)
}

func TestAppRunHelpAfterCommand(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		contains []string
		omits    []string
	}{
		{
			name:     "long",
			args:     []string{"ping", "--help"},
			contains: []string{"--name", "ping [flags]", "--verbose", "--help"},
			omits:    []string{"Commands:"},
		},
		{
			name:     "short",
			args:     []string{"ping", "-h"},
			contains: []string{"--name", "ping [flags]", "--verbose", "--help"},
			omits:    []string{"Commands:"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := ParseOK[App[extraCmds]](t, tc.args...)
			got := test.Stdout(t, func() {
				require.NoError(t, app.Run(t.Context()))
			})
			for _, want := range tc.contains {
				assert.Contains(t, got, want)
			}
			for _, omit := range tc.omits {
				assert.NotContains(t, got, omit)
			}
		})
	}
}

func TestAppInnerHelp(t *testing.T) {
	app := ParseOK[App[extraCmds]](t, "--help")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "ping")
	assert.Contains(t, got, "--verbose")
}

type treeRoot struct {
	foo *fooCmd
}

type fooCmd struct {
	bar  *barCmd
	nick StringArg `long:"nick" help:"foo nick" default:""`
}

func (fooCmd) Description() string {
	return "foo command"
}

type barCmd struct {
	id StringArg `long:"id" help:"bar id"`
}

func (barCmd) Description() string {
	return "bar command"
}

func (*barCmd) Run(context.Context) error { return nil }

func TestAppRunNestedHelp(t *testing.T) {
	app := ParseOK[App[treeRoot]](t, "foo", "--help")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "foo command")
	assert.Contains(t, got, "--nick")
	assert.Contains(t, got, "bar")
	assert.Contains(t, got, "--verbose")
	assert.NotContains(t, got, "--id")

	app = ParseOK[App[treeRoot]](t, "foo", "bar", "--help")
	got = test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "bar command")
	assert.Contains(t, got, "--id")
	assert.Contains(t, got, "--nick")
	assert.Contains(t, got, "--verbose")
}

func TestAppRunNoRunPrintsUsage(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[treeRoot]](t, "foo")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "foo command")
	assert.Contains(t, got, "--nick")
	assert.Contains(t, got, "bar")
	assert.Contains(t, got, "--verbose")
	assert.NotContains(t, got, "--id")
}

type usageLeaf struct{}

func (usageLeaf) Description() string {
	return "leaf usage"
}

func (*usageLeaf) Run(context.Context) error {
	return ErrUsage
}

type usageRoot struct {
	show *usageLeaf
}

func TestAppRunErrUsage(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[usageRoot]](t, "show")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "leaf usage")
	assert.Contains(t, got, "Usage:")
	assert.Contains(t, got, "--verbose")
}

type muteRoot struct {
	mute *muteLeaf
}

type muteLeaf struct {
	name StringArg `long:"name" default:""`
}

func TestAppRunLeafNoRun(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[muteRoot]](t, "mute")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "--name")
	assert.Contains(t, got, "mute [flags]")
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
	test.RestoreSlog(t)

	app := ParseOK[App[rootCmd]](t, "--name", "x")
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

type setupArgs struct {
	called bool
}

func (s *setupArgs) Setup() error {
	s.called = true
	return nil
}

func TestAppCallsArgsSetup(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[setupArgs]](t)
	require.NoError(t, app.Run(t.Context()))
	assert.True(t, app.Args.called)
}

type setupFail struct{}

func (setupFail) Setup() error {
	return ErrInvalidArgument
}

func TestAppArgsSetupError(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[setupFail]](t)
	assert.ErrorIs(t, app.Run(t.Context()), ErrInvalidArgument)
}
