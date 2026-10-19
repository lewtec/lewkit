package term

import (
	"context"
	"fmt"
	"os"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	mux "github.com/lewtec/lewkit/x/term"
)

// connect dials path, starting a detached ubersession when nothing is listening.
func connect(ctx context.Context, path string) (*mux.Client, error) {
	return mux.Connect(ctx, path, func(ctx context.Context) error {
		return spawnServe(ctx, path)
	})
}

// spawnServe re-execs this binary as `term serve` in its own session.
// The child outlives this process. A later attach dials the same socket.
// exec.Start is not used. It kills the command when ctx ends, and it sets
// Setpgid, which rejects a session leader.
func spawnServe(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: %w", mux.ErrSocket, err)
	}
	cmd := execdriver.MustCommand(ctx, exe, "term", "serve", "--socket", path)
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("%w: %w", mux.ErrSocket, err)
	}
	defer null.Close()
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: %w", mux.ErrSocket, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
