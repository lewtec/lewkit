package opener

import (
	"context"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// Start runs name and does not wait for it to exit.
func Start(ctx context.Context, name string, args ...string) error {
	cmd := execdriver.MustRun(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// RequireTool reports ErrIncompatible when name is not on PATH.
func RequireTool(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}
