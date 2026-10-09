//go:build !linux

package gtk

import (
	"context"
	"errors"
)

// ErrUnavailable means GTK 4 is missing or this process cannot open a display.
var ErrUnavailable = errors.New("gtk unavailable")

// Available reports that this process is not Linux.
func Available(context.Context) error { return ErrUnavailable }

// SetPrgname is a no-op where GTK is not the host toolkit.
func SetPrgname(string) {}

// Owned reports whether Ensure has claimed a thread.
func Owned() bool { return false }

// OnOwner reports whether this thread claimed GTK.
func OnOwner() bool { return false }

// Ensure reports that GTK is unavailable.
func Ensure() error { return ErrUnavailable }

// Do reports that GTK is unavailable.
func Do(func()) error { return ErrUnavailable }

// Enqueue reports that GTK is unavailable.
func Enqueue(func()) error { return ErrUnavailable }

// Poll is a no-op where this thread does not own a GTK context.
func Poll() int32 { return 0 }

// SurfaceXID is the X11 window id. Other hosts have none.
func SurfaceXID(uintptr) (uint32, error) { return 0, ErrUnavailable }

// Show reports that GTK is unavailable.
func Show(context.Context, string, string, string) error { return ErrUnavailable }

// Confirm reports that GTK is unavailable.
func Confirm(context.Context, string) (bool, error) { return false, ErrUnavailable }

// Prompt reports that GTK is unavailable.
func Prompt(context.Context, string) (string, bool, error) {
	return "", false, ErrUnavailable
}

// Choose reports that GTK is unavailable.
func Choose(context.Context, string, []string) (int, bool, error) {
	return 0, false, ErrUnavailable
}
