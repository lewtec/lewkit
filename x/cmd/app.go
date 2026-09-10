package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/lewtec/lewkit/x/profile"
)

// App holds process-wide flags shared by lewkit commands.
type App struct {
	verbose    Count     `short:"v" long:"verbose"`
	profileDir StringArg `long:"profile-dir"`
}

// LogLevel is slog.LevelInfo minus 4 for each -v/--verbose count.
func (a App) LogLevel() slog.Level {
	return slog.LevelInfo - slog.Level(4*a.verbose.Value())
}

// Run sets the default slog level from -v/--verbose.
// If --profile-dir is set, it records pprof profiles until ctx is done.
func (a *App) Run(ctx context.Context) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: a.LogLevel()})))
	if dir := a.profileDir.Value(); dir != "" {
		p := profile.NewProfile(dir)
		return p.Run(ctx)
	}
	return nil
}
