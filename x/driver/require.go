package driver

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type envKey struct{}

// WithEnv stores KEY=VALUE pairs on ctx for GetEnv.
func WithEnv(ctx context.Context, env []string) context.Context {
	return context.WithValue(ctx, envKey{}, env)
}

// GetEnv reads key from ctx env when present, otherwise the process environment.
func GetEnv(ctx context.Context, key string) string {
	if env, ok := ctx.Value(envKey{}).([]string); ok {
		for _, e := range env {
			k, v, found := strings.Cut(e, "=")
			if found && k == key {
				return v
			}
		}
	}
	return os.Getenv(key)
}

// IsTermux reports whether TERMUX_VERSION is set.
func IsTermux() bool {
	return os.Getenv("TERMUX_VERSION") != ""
}

// RequireTermux returns ErrIncompatible when not running under Termux.
func RequireTermux() error {
	if IsTermux() {
		return nil
	}
	return fmt.Errorf("%w: not running in Termux", ErrIncompatible)
}

// RequireEnv returns ErrIncompatible when key is unset or empty.
func RequireEnv(ctx context.Context, key string) error {
	if GetEnv(ctx, key) != "" {
		return nil
	}
	return fmt.Errorf("%w: %s not set", ErrIncompatible, key)
}

// RequireDisplay returns ErrIncompatible when neither DISPLAY nor WAYLAND_DISPLAY is set.
func RequireDisplay(ctx context.Context) error {
	if GetEnv(ctx, "DISPLAY") == "" && GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: neither DISPLAY nor WAYLAND_DISPLAY set", ErrIncompatible)
	}
	return nil
}

// WrapIncompatible returns nil when err is nil.
// The text is err's, and errors.Is matches ErrIncompatible only.
func WrapIncompatible(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrIncompatible, err)
}

// RequireAnyEnv returns ErrIncompatible when none of the keys are set.
func RequireAnyEnv(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if GetEnv(ctx, key) != "" {
			return nil
		}
	}
	return fmt.Errorf("%w: none of %s set", ErrIncompatible, strings.Join(keys, ", "))
}
