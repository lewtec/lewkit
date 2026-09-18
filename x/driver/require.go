package driver

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type envKey struct{}

// WithEnv stores KEY=VALUE pairs on ctx for GetEnv.
func WithEnv(ctx context.Context, env []string) context.Context {
	return context.WithValue(ctx, envKey{}, env)
}

// GetEnv reads key from ctx env when present, otherwise the process environment.
func GetEnv(ctx context.Context, key string) string {
	if ctx != nil {
		if env, ok := ctx.Value(envKey{}).([]string); ok {
			for _, e := range env {
				k, v, found := strings.Cut(e, "=")
				if found && k == key {
					return v
				}
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

// RequireGOOS returns ErrIncompatible when runtime.GOOS is not name.
func RequireGOOS(name string) error {
	if runtime.GOOS == name {
		return nil
	}
	return fmt.Errorf("%w: not %s", ErrIncompatible, name)
}

// RequireEnv returns ErrIncompatible when key is unset or empty.
func RequireEnv(ctx context.Context, key string) error {
	if GetEnv(ctx, key) != "" {
		return nil
	}
	return fmt.Errorf("%w: %s not set", ErrIncompatible, key)
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
