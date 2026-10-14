package opener

import (
	"context"
	"errors"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

var (
	ErrStart = errors.New("")
)

// Start runs name and does not wait for it to exit.
func Start(ctx context.Context, name string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	if err := execdriver.Start(ctx, cmd); err != nil {
		return fmt.Errorf("%w%s: %w", ErrStart, name, err)
	}
	return nil
}

// RequireTool reports ErrIncompatible when name is not on PATH.
func RequireTool(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}
