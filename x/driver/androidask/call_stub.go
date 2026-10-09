//go:build !android

package androidask

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

// Platform reports that this process has no Java VM.
func Platform(context.Context) error {
	return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

// Call reports that this process has no Java VM.
func Call(context.Context, string, ...any) (string, error) {
	return "", fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
