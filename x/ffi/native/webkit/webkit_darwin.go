//go:build darwin

package webkit

import (
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means WebKit.framework could not be opened.
var ErrUnavailable = errors.New("webkit unavailable")

// Load opens Cocoa and WebKit. Later Objective-C calls use those frameworks.
func Load() error {
	if _, err := native.Open("/System/Library/Frameworks/Cocoa.framework/Cocoa", native.Global|native.Lazy); err != nil {
		return fmt.Errorf("%w: cocoa: %v", ErrUnavailable, err)
	}
	if _, err := native.Open("/System/Library/Frameworks/WebKit.framework/WebKit", native.Global|native.Lazy); err != nil {
		return fmt.Errorf("%w: webkit: %v", ErrUnavailable, err)
	}
	return nil
}
