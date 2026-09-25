package terminal

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
)

// RequireBinary returns ErrIncompatible when name is not on PATH.
func RequireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

// RequireEnvBinary returns ErrIncompatible when key is unset or name is not on PATH.
func RequireEnvBinary(ctx context.Context, key, name string) error {
	if err := driver.RequireEnv(ctx, key); err != nil {
		return err
	}
	return RequireBinary(name)
}
