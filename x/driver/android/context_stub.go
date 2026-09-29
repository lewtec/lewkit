//go:build !android || !cgo

package android

import (
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

// Context returns the application context stored on lewkit.Host.
// The caller releases the reference.
func Context() (*jni.Ref, error) {
	return nil, fmt.Errorf("%w: application context", driver.ErrUnavailable)
}
