//go:build !darwin

package webkit

import (
	"errors"
	"fmt"
)

// ErrUnavailable means WebKit.framework is not loaded.
var ErrUnavailable = errors.New("webkit unavailable")

// Load reports that WebKit.framework is a macOS library.
func Load() error {
	return fmt.Errorf("%w: not darwin", ErrUnavailable)
}
