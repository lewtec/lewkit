package terminal

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// RequireBinary returns ErrIncompatible when name is not on PATH.
func RequireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

// RequireEnvBinary returns ErrIncompatible when key is unset or name is not on PATH.
func RequireEnvBinary(ctx context.Context, key, name string) error {
	if err := driver.RequireEnv(ctx, key); err != nil {
		return err
	}
	return RequireBinary(ctx, name)
}
