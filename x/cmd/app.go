package cmd

import (
	"context"
	"errors"
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
	verbose Count     `short:"v" long:"verbose" help:"log verbosity" ctx:"verbose"`
	pprof   StringArg `long:"pprof" help:"pprof directory or listen address" default:"" ctx:"pprof"`
	help    Flag      `short:"h" long:"help" help:"show help" ctx:"help"`
	version Flag      `long:"version" help:"print version" ctx:"version"`
	Args    T         `flatten:""`
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

// Setup sets the default slog level, calls Args.Setup when T has that
// method, and starts the profiler in a goroutine when --pprof is set.
// The profiler stops when ctx is done.
func (a *App[T]) Setup(ctx context.Context) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: a.LogLevel()})))
	if s, ok := any(&a.Args).(interface{ Setup() error }); ok {
		if err := s.Setup(); err != nil {
			return err
		}
	}
	destination := a.pprof.Value()
	if destination == "" {
		return nil
	}
	p := newProfile(destination)
	go func() {
		if err := p.Run(ctx); err != nil {
			slog.Error(err.Error())
		}
	}()
	return nil
}

func newProfile(destination string) profile.Profile {
	var listen AddrArg
	if err := listen.Parse(destination); err == nil {
		return profile.Address(listen.Value())
	}
	return profile.Directory(destination)
}

// Run prints help or version when asked, then Setup, then T's selected
// command (or T itself) if it has Run(ctx) error. Missing Run, or a
// Run that returns ErrUsage, prints that command's usage and succeeds.
func (a *App[T]) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	ctx = withValues(ctx)
	bind(ctx, reflect.ValueOf(a).Elem())
	switch {
	case Get[bool](ctx, "help"):
		return a.printUsage()
	case Get[bool](ctx, "version"):
		return release.PrintVersion(os.Stdout)
	}
	if err := a.Setup(ctx); err != nil {
		return err
	}
	err := runSelected(ctx, reflect.ValueOf(&a.Args).Elem())
	if errors.Is(err, ErrUsage) {
		return a.printUsage()
	}
	return err
}

func (a *App[T]) printUsage() error {
	text, err := usageSelected(reflect.ValueOf(a).Elem(), filepath.Base(os.Args[0]))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, text)
	return err
}
