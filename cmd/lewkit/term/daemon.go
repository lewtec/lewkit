package term

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/daemon"
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
func spawnServe(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := daemon.Self(ctx, "term", "serve", "--socket", path); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %w", mux.ErrSocket, err)
	}
	return nil
}
