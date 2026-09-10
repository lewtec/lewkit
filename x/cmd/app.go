package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/profile"
	"github.com/lewtec/lewkit/x/release"
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

// Run handles --help/--version, sets the default slog level, and starts pprof
// when --profile-dir is set (until ctx is done).
func (a *App) Run(ctx context.Context) error {
	switch {
	case a.help.Value():
		text, err := Usage[App](filepath.Base(os.Args[0]))
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, text)
		return err
	case a.version.Value() || a.versionCmd != nil:
		_, err := fmt.Fprintln(os.Stdout, release.Version())
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: a.LogLevel()})))
	if dir := a.profileDir.Value(); dir != "" {
		p := profile.NewProfile(dir)
		return p.Run(ctx)
	}
	return nil
}
