// Package daemon starts a process that outlives the caller.
//
// Spawn starts a program in its own session. Self re-execs this binary.
// Standard streams are the null device. The call returns after the process
// has started. A goroutine waits so an exit is reaped, and the process
// keeps running after the caller returns.
// exec.Start is not used. It kills the command when the context ends, and
// it sets Setpgid, which rejects a session leader.
package daemon

import (
	"context"
	"fmt"
	"os"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"github.com/lewtec/lewkit/x/errs"
)

// ErrSpawn means the detached process did not start.
var ErrSpawn = errs.Wrap(errs.ErrUnavailable, "spawn daemon")

// Spawn starts name with args in its own session.
// The caller does not wait for the process to exit.
func Spawn(ctx context.Context, name string, args ...string) error {
	if ctx == nil {
		return errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("%w: empty program", ErrSpawn)
	}
	cmd := execdriver.MustCommand(ctx, name, args...)
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSpawn, err)
	}
	defer null.Close()
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: %w", ErrSpawn, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Self re-execs this binary with args.
func Self(ctx context.Context, args ...string) error {
	if ctx == nil {
		return errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSpawn, err)
	}
	return Spawn(ctx, exe, args...)
}
