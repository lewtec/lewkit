package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/lewtec/lewkit/x/profile"
	"github.com/lewtec/lewkit/x/release"
)

// None is an App with no extra flags or commands.
type None struct{}

// App wraps process-wide flags around T, the rest of the command spec.
type App[T any] struct {
	verbose    Count     `short:"v" long:"verbose" help:"log verbosity"`
	profileDir StringArg `long:"profile-dir" help:"write pprof profiles here" default:""`
	help       Flag      `short:"h" long:"help" help:"show help"`
	version    Flag      `long:"version" help:"print version"`
	Args       T         `flatten:""`
}

// LogLevel is slog.LevelInfo minus 4 for each -v/--verbose count.
func (a App[T]) LogLevel() slog.Level {
	return slog.LevelInfo - slog.Level(4*a.verbose.Value())
}

func (a App[T]) Help() bool {
	return a.help.Value()
}

func (a App[T]) Description() string {
	return descriptionOf[T]()
}

func (a App[T]) WantVersion() bool {
	return a.version.Value()
}

// Setup sets the default slog level and starts the profiler in a goroutine
// when --profile-dir is set. The profiler stops when ctx is done.
func (a *App[T]) Setup(ctx context.Context) error {
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

// Run prints help or version when asked, then Setup, then T's selected
// command (or T itself) if it has Run(ctx) error.
func (a *App[T]) Run(ctx context.Context) error {
	switch {
	case a.Help():
		text, err := Usage[App[T]](filepath.Base(os.Args[0]))
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, text)
		return err
	case a.WantVersion():
		return release.PrintVersion(os.Stdout)
	}
	if err := a.Setup(ctx); err != nil {
		return err
	}
	return runSelected(ctx, reflect.ValueOf(&a.Args).Elem())
}
