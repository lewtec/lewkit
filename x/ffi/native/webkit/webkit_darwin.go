//go:build darwin

package webkit

import (
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means WebKit.framework could not be opened.
var ErrUnavailable = errs.Wrap(errs.ErrUnavailable, "webkit unavailable")

// Load opens the host UI framework and WebKit.
// Later Objective-C calls use those frameworks.
// iOS has no Cocoa.framework, so that host opens UIKit.
func Load() error {
	ui := "/System/Library/Frameworks/Cocoa.framework/Cocoa"
	uiName := "cocoa"
	if runtime.GOOS == "ios" {
		ui = "/System/Library/Frameworks/UIKit.framework/UIKit"
		uiName = "uikit"
	}
	if _, err := native.Open(ui, native.Global|native.Lazy); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, uiName, err)
	}
	if _, err := native.Open("/System/Library/Frameworks/WebKit.framework/WebKit", native.Global|native.Lazy); err != nil {
		return fmt.Errorf("%w: webkit: %v", ErrUnavailable, err)
	}
	return nil
}
