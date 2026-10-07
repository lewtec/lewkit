// Package exec runs host programs.
//
// Command and MustCommand take a context to select the driver. That context
// is not stored on the command.
// Run, Start, Output, and Wait take a command and a context: it cancels the
// process and, when stderr is still unset, attaches the stderr hook.
// RunProgram and OutputString take a program name and args.
// taskgroup registers that hook with the line writer.
// Stdout stays unset so Output can capture it.
// Import prelude or native so the host driver is registered.
// Termux stays in modot.
package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	// ErrNotFound means the executable is not on PATH.
	ErrNotFound = errors.New("executable not found")
	// ErrNilCommand means Start was given no command.
	ErrNilCommand = errors.New("nil command")
	// ErrStdoutSet means Output was called with Stdout already set.
	ErrStdoutSet = errors.New("stdout already set")
)

// stderrHook, when set, supplies Stderr if it is still nil at start.
// Unset means os.Stderr. taskgroup registers the line writer.
var stderrHook atomic.Value // func(context.Context) io.Writer

// SetStderr registers the writer used when Stderr is still nil.
// A nil fn restores os.Stderr. The last call wins.
func SetStderr(fn func(context.Context) io.Writer) {
	if fn == nil {
		stderrHook.Store(func(context.Context) io.Writer { return os.Stderr })
		return
	}
	stderrHook.Store(fn)
}

func stderrWriter(ctx context.Context) io.Writer {
	fn, ok := stderrHook.Load().(func(context.Context) io.Writer)
	if !ok || fn == nil {
		return os.Stderr
	}
	return fn(ctx)
}

// Driver builds commands and resolves executables for this host.
// Command does not start the process. The facade passes the caller context
// to Get and does not store it on the command.
type Driver interface {
	Command(name string, args ...string) *exec.Cmd
	Which(ctx context.Context, name string) (string, error)
}

// Command returns a command from the selected driver.
// ctx selects the driver. It is not stored on the command.
// The process is not started and streams are left unset.
func Command(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, errors.New("exec: nil context")
	}
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return nil, err
	}
	return d.Command(name, args...), nil
}

// MustCommand is Command, or a raw command when no driver is registered.
// A nil context panics.
func MustCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	if ctx == nil {
		panic("exec: nil context")
	}
	cmd, err := Command(ctx, name, args...)
	if err != nil {
		slog.WarnContext(ctx, "exec driver unavailable", "name", name, "error", err)
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
		return ErrNilCommand
	}
	if ctx == nil {
		return errors.New("exec: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd.Stderr == nil {
		cmd.Stderr = stderrWriter(ctx)
	}
	// A shell's children inherit the stdout and stderr pipes. Killing only
	// the shell leaves those children holding the pipes, and Wait never
	// returns. The command runs in its own group so cancel reaches them.
	prepareCancel(cmd)
	slog.DebugContext(ctx, "exec", "path", cmd.Path, "args", cmd.Args)
	if err := cmd.Start(); err != nil {
		return err
	}
	context.AfterFunc(ctx, func() { cancelProcess(cmd) })
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
		return nil, ErrStdoutSet
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := Run(ctx, cmd)
	return buf.Bytes(), err
}

// RunProgram runs name with args. A failure names the program.
func RunProgram(ctx context.Context, name string, args ...string) error {
	if ctx == nil {
		return errors.New("exec: nil context")
	}
	if err := Run(ctx, MustCommand(ctx, name, args...)); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// OutputString runs name with args and returns stdout text.
// A failure names the program.
func OutputString(ctx context.Context, name string, args ...string) (string, error) {
	if ctx == nil {
		return "", errors.New("exec: nil context")
	}
	out, err := Output(ctx, MustCommand(ctx, name, args...))
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
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
