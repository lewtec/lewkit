//go:build !windows

package webview2

import (
	"errors"
	"fmt"
)

// ErrUnavailable means the WebView2 loader is not loaded.
var ErrUnavailable = errors.New("webview2 unavailable")

// Available reports that WebView2 is a Windows library.
func Available() error {
	return fmt.Errorf("%w: not windows", ErrUnavailable)
}
