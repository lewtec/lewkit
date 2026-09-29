//go:build android && cgo

package android

import (
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

// Context returns the application context stored on lewkit.Host.
// The caller releases the reference.
func Context() (*jni.Ref, error) {
	app, err := Ref(jni.StaticField("lewkit.Host", "app"))
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("%w: application context", driver.ErrUnavailable)
	}
	return app, nil
}
