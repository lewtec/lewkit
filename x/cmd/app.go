package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/lewtec/lewkit/x/profile"
)

// App holds process-wide flags shared by lewkit commands.
type App struct {
	verbose    Count       `short:"v" long:"verbose" help:"log verbosity"`
	profileDir StringArg   `long:"profile-dir" help:"write pprof profiles here"`
	help       Flag        `short:"h" long:"help" help:"show help"`
	version    Flag        `long:"version" help:"print version"`
	versionCmd *versionCmd `cmd:"version" help:"print version"`
}

type versionCmd struct{}

// LogLevel is slog.LevelInfo minus 4 for each -v/--verbose count.
func (a App) LogLevel() slog.Level {
	return slog.LevelInfo - slog.Level(4*a.verbose.Value())
}

func (a App) Help() bool {
	return a.help.Value()
}

func (a App) WantVersion() bool {
	return a.version.Value() || a.versionCmd != nil
}

// Setup sets the default slog level and starts the profiler in a goroutine
// when --profile-dir is set. The profiler stops when ctx is done.
func (a *App) Setup(ctx context.Context) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: a.LogLevel()})))
	if dir := a.profileDir.Value(); dir == "" {
		return nil
	}
	p := profile.NewProfile(a.profileDir.Value())
	go func() {
		if err := p.Run(ctx); err != nil {
			slog.Error(err.Error())
		}
	}()
	return nil
}

// Run is Run[App]. Types that embed App should call Run[T] with the outer
// value so help and command dispatch see the full spec.
func (a *App) Run(ctx context.Context) error {
	return Run(ctx, *a)
}
