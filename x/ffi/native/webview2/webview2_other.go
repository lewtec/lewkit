//go:build !windows

package webview2

import (
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
)

// ErrUnavailable means the Edge WebView2 runtime is not available.
var ErrUnavailable = errs.Wrap(errs.ErrUnavailable, "webview2 unavailable")

// Available reports that WebView2 is a Windows library.
func Available() error {
	return fmt.Errorf("%w: not windows", ErrUnavailable)
}
