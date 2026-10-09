//go:build !linux

package gtk

import (
	"context"
	"errors"
)

// ErrUnavailable means GTK 4 is missing or this process cannot open a display.
var ErrUnavailable = errors.New("gtk unavailable")

// Available reports that this process is not Linux.
func Available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}

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
func Show(ctx context.Context, _, _, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}

// Confirm reports that GTK is unavailable.
func Confirm(ctx context.Context, _ string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return false, ErrUnavailable
}

// Prompt reports that GTK is unavailable.
func Prompt(ctx context.Context, _ string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	return "", false, ErrUnavailable
}

// Choose reports that GTK is unavailable.
func Choose(ctx context.Context, _ string, _ []string) (int, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	return 0, false, ErrUnavailable
}
