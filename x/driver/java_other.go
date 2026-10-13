//go:build !android

package driver

import "fmt"

// JavaVM reports ErrIncompatible when this process has no Java VM.
func JavaVM() error {
	return fmt.Errorf("%w: no Java VM", ErrIncompatible)
}
