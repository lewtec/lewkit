//go:build !darwin

package webkit

import (
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
)

// ErrUnavailable means WebKit.framework is not loaded.
var ErrUnavailable = errs.Wrap(errs.ErrUnavailable, "webkit unavailable")

// Load reports that WebKit.framework is a macOS library.
func Load() error {
	return fmt.Errorf("%w: not darwin", ErrUnavailable)
}
