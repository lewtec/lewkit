// Package gtk shows a message with a GTK 4 window.
package gtk

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	"github.com/lewtec/lewkit/x/driver/thread"
	gtklib "github.com/lewtec/lewkit/x/ffi/native/gtk"

	_ "github.com/lewtec/lewkit/x/driver/thread/std"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_gtk" }
func (factory) Name() string { return "GTK alert" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS("linux"); err != nil {
		return err
	}
	if driver.GetEnv(ctx, "DISPLAY") == "" && driver.GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: neither DISPLAY nor WAYLAND_DISPLAY set", driver.ErrIncompatible)
	}
	if err := gtklib.Available(ctx); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return present(ctx, func() error {
		return gtklib.Show(ctx, n.Title, n.Message, n.Style)
	})
}

func present(ctx context.Context, fn func() error) error {
	if gtklib.Owned() && !gtklib.OnOwner() {
		return fn()
	}
	if bound, on := uiThread(); bound && !on {
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

func uiThread() (bound, on bool) {
	defer func() { _ = recover() }()
	if !thread.Bound() {
		return false, false
	}
	return true, thread.On()
}

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
