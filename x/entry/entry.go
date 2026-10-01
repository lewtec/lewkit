// Package entry is the process startup shared by apps and commands.
//
// Main installs the signal context, binds the UI thread, and runs work
// inside one taskgroup session and its progress view. Run is the same
// sequence when the caller already owns the process lifetime.
package entry

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/lewtec/lewkit/x/logging"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

func slogOut() io.Writer {
	if w := logging.Logcat(); w != nil {
		return w
	}
	return os.Stderr
}

// Main runs fn as the process. A non-nil error is logged and the process exits 1.
func Main(fn func(context.Context) error) {
	slog.SetDefault(slog.New(logging.NewHandler(slogOut(), &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := Run(ctx, fn); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

// after runs once the progress view has stopped.
// The context is the one thread.Run passed in. Session.Wait does not cancel it.
var after func(context.Context) error

// After registers fn to run after the progress view stops.
// fn receives the context Run held for the UI thread.
// A later call replaces fn. Nil clears it.
func After(fn func(context.Context) error) {
	after = fn
}

// Run binds the UI thread, attaches one taskgroup session, and shows its progress.
// A session already on ctx is kept.
// After runs when fn and the session succeed, before this call returns.
func Run(ctx context.Context, fn func(context.Context) error) error {
	if w := logging.Logcat(); w != nil {
		slog.SetDefault(slog.New(logging.NewHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return thread.Run(ctx, func(ctx context.Context) error {
		parent := ctx
		var err error
		if taskgroup.FromContext(ctx) == nil {
			var session *taskgroup.Session
			session, ctx = taskgroup.New(ctx, taskgroup.DefaultLimits())
			err = progress.Run(session, ctx, fn)
		} else {
			err = progress.Run(taskgroup.FromContext(ctx), ctx, fn)
		}
		follow := after
		after = nil
		if err != nil {
			return err
		}
		if follow == nil {
			return nil
		}
		return follow(parent)
	})
}
