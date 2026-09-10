package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"

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
