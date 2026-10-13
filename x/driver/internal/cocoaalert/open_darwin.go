//go:build darwin

// Package cocoaalert opens Cocoa once for dialog drivers.
// A missing framework is [ErrAlert].
package cocoaalert

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const framework = "/System/Library/Frameworks/Cocoa.framework/Cocoa"

// ErrAlert means the Cocoa framework did not open, or a dialog view is missing.
var ErrAlert = errors.New("alert")

var (
	once    sync.Once
	openErr error
)

// Open loads the Cocoa framework. Later calls return the first result.
func Open() error {
	once.Do(func() {
		if _, err := native.Open(framework, native.Global|native.Lazy); err != nil {
			openErr = fmt.Errorf("%w: %w", ErrAlert, err)
		}
	})
	return openErr
}
