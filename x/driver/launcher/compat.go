package launcher

import (
	"context"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// RequireBinary returns ErrIncompatible when name is not on PATH.
func RequireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

// RequireEnvBinary returns ErrIncompatible when key is unset or name is not on PATH.
func RequireEnvBinary(ctx context.Context, key, name string) error {
	return execdriver.RequireEnvBinary(ctx, key, name)
}

// RequireDisplayBinary returns ErrIncompatible when neither DISPLAY nor
// WAYLAND_DISPLAY is set, or when name is not on PATH.
func RequireDisplayBinary(ctx context.Context, name string) error {
	return execdriver.RequireDisplayBinary(ctx, name)
}
