//go:build darwin && !ios

package uikit

import (
	"fmt"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
)

func openHost(objc.ID) (objc.ID, error) {
	return 0, fmt.Errorf("%w: no window", window.ErrInit)
}

func surfaceClass() objc.Class {
	return objc.GetClass("UIView")
}
