//go:build !android

package androidask

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Platform reports that this process has no Java VM.
func Platform(context.Context) error {
	return driver.JavaVM()
}

// Call reports that this process has no Java VM.
func Call(context.Context, string, ...any) (string, error) {
	return "", driver.JavaVM()
}
