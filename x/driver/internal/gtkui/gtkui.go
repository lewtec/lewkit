// Package gtkui is the shared GTK 4 gate for launcher and messagebox dialogs.
// A missing display or library is [driver.ErrIncompatible].
// Zenity uses the same display check and does not open GTK.
package gtkui

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/thread"
	gtklib "github.com/lewtec/lewkit/x/ffi/native/gtk"
)

// Require reports whether this process can open a GTK 4 dialog.
func Require(ctx context.Context) error {
	if err := driver.ForGOOS("linux"); err != nil {
		return err
	}
	if err := driver.RequireDisplay(ctx); err != nil {
		return err
	}
	if err := gtklib.Available(ctx); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

// Run calls fn on the thread that owns GTK.
// The first dialog claims the UI thread when one is bound.
func Run(ctx context.Context, fn func() error) error {
	if gtklib.Owned() && !gtklib.OnOwner() {
		return fn()
	}
	if bound, on := thread.State(); bound && !on {
		done := make(chan error, 1)
		thread.Go(func() { done <- fn() })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return fn()
}
