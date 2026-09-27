package launcher

import (
	"context"
	"fmt"

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

// RequireDisplayBinary returns ErrIncompatible when neither DISPLAY nor
// WAYLAND_DISPLAY is set, or when name is not on PATH.
func RequireDisplayBinary(ctx context.Context, name string) error {
	if driver.GetEnv(ctx, "DISPLAY") == "" && driver.GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: neither DISPLAY nor WAYLAND_DISPLAY set", driver.ErrIncompatible)
	}
	return RequireBinary(ctx, name)
}
