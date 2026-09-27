// Package entry is the process startup shared by apps and commands.
//
// Main installs the signal context, binds the UI thread, and runs work
// inside one taskgroup session and its progress view. Run is the same
// sequence when the caller already owns the process lifetime.
package entry

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/lewtec/lewkit/x/logging"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

// Main runs fn as the process. A non-nil error is logged and the process exits 1.
func Main(fn func(context.Context) error) {
	slog.SetDefault(slog.New(logging.NewHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := Run(ctx, fn); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

// Run binds the UI thread, attaches one taskgroup session, and shows its progress.
// A session already on ctx is kept.
func Run(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return thread.Run(ctx, func(ctx context.Context) error {
		if taskgroup.FromContext(ctx) == nil {
			var session *taskgroup.Session
			session, ctx = taskgroup.New(ctx, taskgroup.DefaultLimits())
			return progress.Run(session, ctx, fn)
		}
		return progress.Run(taskgroup.FromContext(ctx), ctx, fn)
	})
}
