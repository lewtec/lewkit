// Package entry is the process startup shared by apps and commands.
//
// Main installs the signal context, binds the UI thread, and runs work
// inside one taskgroup session and its progress view. The process root
// is the context main passes in. Run is the same sequence when the
// caller already owns the process lifetime.
package entry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/lewtec/lewkit/report"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	_ "github.com/lewtec/lewkit/x/driver/messagebox/prelude"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/lewtec/lewkit/x/logging"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

// guard turns a panic into an error while the process is an app.
// A command still panics.
func guard(fn func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) (err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if !failureVisible() {
				panic(recovered)
			}
			err = fmt.Errorf("panic: %v", recovered)
		}()
		return fn(ctx)
	}
}

func slogOut() io.Writer {
	if w := logging.Logcat(); w != nil {
		return w
	}
	return os.Stderr
}

// Main runs fn as the process. parent is the root from main.
// A non-nil error is logged, reported, and the process exits 1.
func Main(parent context.Context, fn func(context.Context) error) {
	MainFrom(parent, fn)
}

// MainFrom is Main with parent's context values kept on the signal context.
// Pool caps from taskgroup.WithLimits apply when Run starts the session.
func MainFrom(parent context.Context, fn func(context.Context) error) {
	prepareHost()
	slog.SetDefault(slog.New(logging.NewHandler(slogOut(), &slog.HandlerOptions{Level: slog.LevelInfo})))
	if parent == nil {
		slog.Error("entry: nil context")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt)
	defer stop()
	if err := Run(ctx, fn); err != nil {
		slog.Error(err.Error())
		report.Report(err)
		showFailure(ctx, err)
		os.Exit(1)
	}
}

// failureVisible is app mode, or a Windows GUI executable with no console.
func failureVisible() bool {
	return driver.AppMode() || windowsGUI()
}

// showFailure is the escape hatch when the process has no terminal.
// The error is shown in a message box before the process exits.
func showFailure(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if ctx == nil {
		panic("entry: nil context")
	}
	if !failureVisible() {
		return
	}
	box, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	if showErr := messagebox.Show(box, release.Name(), err.Error()); showErr != nil {
		NotifyFail(err.Error())
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
		return fmt.Errorf("entry: nil context")
	}
	return thread.Run(ctx, func(ctx context.Context) (err error) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if !failureVisible() {
				panic(rec)
			}
			err = fmt.Errorf("panic: %v", rec)
			slog.Error(err.Error())
		}()
		parent := ctx
		if taskgroup.FromContext(ctx) == nil {
			limits := taskgroup.DefaultLimits()
			if chosen, ok := taskgroup.LimitsFrom(ctx); ok {
				limits = chosen
			}
			var session *taskgroup.Session
			session, ctx = taskgroup.New(ctx, limits)
			err = progress.Run(session, ctx, guard(fn))
		} else {
			err = progress.Run(taskgroup.FromContext(ctx), ctx, guard(fn))
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
