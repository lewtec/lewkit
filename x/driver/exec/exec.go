// Package exec runs host programs.
//
// Command builds a command and does not take a context.
// Run, Start, Output, and Wait take the context: it cancels the process
// and, when stderr is still unset, attaches the taskgroup line writer.
// Stdout stays unset so Output can capture it.
// Import prelude or native so the host driver is registered.
// Termux stays in modot.
package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// ErrNotFound means the executable is not on PATH.
var ErrNotFound = errors.New("executable not found")

// Driver builds commands and resolves executables for this host.
// Command does not take a context and does not start the process.
type Driver interface {
	Command(name string, args ...string) *exec.Cmd
	Which(ctx context.Context, name string) (string, error)
}

// Command returns a command from the selected driver.
// The process is not started and streams are left unset.
func Command(name string, args ...string) (*exec.Cmd, error) {
	d, err := driver.Get[Driver](context.Background())
	if err != nil {
		return nil, err
	}
	return d.Command(name, args...), nil
}

// MustCommand is Command, or a raw command when no driver is registered.
func MustCommand(name string, args ...string) *exec.Cmd {
	cmd, err := Command(name, args...)
	if err != nil {
		slog.Warn("exec driver unavailable", "name", name, "error", err)
		return exec.Command(name, args...)
	}
	return cmd
}

// Run starts cmd and waits. ctx kills the process when it is cancelled.
func Run(ctx context.Context, cmd *exec.Cmd) error {
	if err := Start(ctx, cmd); err != nil {
		return err
	}
	return Wait(ctx, cmd)
}

// Start starts cmd. ctx kills the process when it is cancelled.
func Start(ctx context.Context, cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("nil command")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd.Stderr == nil {
		cmd.Stderr = taskgroup.LineWriterFrom(ctx)
	}
	slog.DebugContext(ctx, "exec", "path", cmd.Path, "args", cmd.Args)
	if err := cmd.Start(); err != nil {
		return err
	}
	context.AfterFunc(ctx, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return nil
}

// Wait waits for cmd. A cancelled ctx wins over the process exit error.
func Wait(ctx context.Context, cmd *exec.Cmd) error {
	err := cmd.Wait()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

// Output runs cmd and returns stdout. Stdout must still be unset.
func Output(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if cmd.Stdout != nil {
		return nil, errors.New("stdout already set")
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := Run(ctx, cmd)
	return buf.Bytes(), err
}

// Which resolves name with the selected driver, or LookPath when none is registered.
func Which(ctx context.Context, name string) (string, error) {
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return lookPath(name)
	}
	return d.Which(ctx, name)
}

// IsBinaryAvailable reports whether Which finds name.
func IsBinaryAvailable(ctx context.Context, name string) bool {
	_, err := Which(ctx, name)
	return err == nil
}

// RequireBinary returns ErrIncompatible when name is not on PATH.
func RequireBinary(ctx context.Context, name string) error {
	if IsBinaryAvailable(ctx, name) {
		return nil
	}
	return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
}

func lookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return path, nil
}
