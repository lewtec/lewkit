// Package androidcall is the process hook the Android JNI host calls.
// x/entry installs the functions. x/entry/android calls them. Neither
// package imports the other.
package androidcall

import (
	"context"
	"errors"
)

var (
	ErrRunEntryNoApp = errors.New("entry: no app")
)

var (
	// PointerFunc receives a touch sample. action is 0 down, 1 up, 2 move.
	PointerFunc func(x, y, action int)
	// ResizeFunc receives the new surface size.
	ResizeFunc func(width, height int)
	// LostFunc runs when the host destroys the current surface.
	LostFunc func()
	// FontFunc receives the system text scale.
	FontFunc func(float64)
	// InsetsFunc receives the obscured edges and the surface size.
	InsetsFunc func(left, top, right, bottom, width, height int)
	// RunFunc runs the app registered with entry.Bind.
	RunFunc func(context.Context) error
	// FailFunc reports a startup failure to the Android host.
	FailFunc func(string)
)

// Pointer delivers one touch sample.
func Pointer(x, y, action int) {
	if PointerFunc != nil {
		PointerFunc(x, y, action)
	}
}

// Resize delivers a new surface size.
func Resize(width, height int) {
	if ResizeFunc != nil {
		ResizeFunc(width, height)
	}
}

// Lost delivers surface destruction.
func Lost() {
	if LostFunc != nil {
		LostFunc()
	}
}

// Font delivers the system text scale.
func Font(scale float64) {
	if FontFunc != nil {
		FontFunc(scale)
	}
}

// Insets delivers the obscured edges.
func Insets(left, top, right, bottom, width, height int) {
	if InsetsFunc != nil {
		InsetsFunc(left, top, right, bottom, width, height)
	}
}

// Run runs the registered app. No registration is "entry: no app".
func Run(ctx context.Context) error {
	if RunFunc == nil {
		return ErrRunEntryNoApp
	}
	return RunFunc(ctx)
}

// Fail reports a startup failure.
func Fail(message string) {
	if FailFunc != nil {
		FailFunc(message)
	}
}
