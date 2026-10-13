//go:build android

package driver

import (
	"fmt"

	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

// JavaVM reports ErrIncompatible when this process has no Java VM.
func JavaVM() error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", ErrIncompatible)
	}
	return nil
}
