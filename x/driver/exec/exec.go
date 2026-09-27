// Package exec runs host programs.
//
// Stderr goes to the taskgroup live row when ctx carries a session.
// Stdout stays unset so Cmd.Output still captures it.
// Import prelude or native so the host driver is registered.
// Termux stays in modot.
package exec

import (
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

// Driver creates commands and resolves executables for this host.
type Driver interface {
	Run(ctx context.Context, name string, args ...string) *exec.Cmd
	Which(ctx context.Context, name string) (string, error)
}

// Run returns a command from the selected driver.
// Stderr is a session line writer. Stdout is left unset.
func Run(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	slog.DebugContext(ctx, "exec", "name", name, "args", args)
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return nil, err
	}
	cmd := d.Run(ctx, name, args...)
	attachDefaultWriters(ctx, cmd)
	return cmd, nil
}

// MustRun is Run, or a raw command when no driver is registered.
// The fallback still attaches the line writer.
func MustRun(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd, err := Run(ctx, name, args...)
	if err != nil {
		slog.WarnContext(ctx, "exec driver unavailable", "name", name, "error", err)
		cmd = exec.CommandContext(ctx, name, args...)
		attachDefaultWriters(ctx, cmd)
	}
	return cmd
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

func attachDefaultWriters(ctx context.Context, cmd *exec.Cmd) {
	cmd.Stderr = taskgroup.LineWriterFrom(ctx)
}
